package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	authEmail    string
	authPassword string
	authMode     string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Authenticate s0 CLI using OIDC device login or built-in account login.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with server-selected provider",
	Long:  `Login using the first provider returned by /auth/providers (server-side provider selection).`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := getConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		profileName := cfg.GetActiveProfile()
		p, err := cfg.GetProfile(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading profile: %v\n", err)
			os.Exit(1)
		}
		baseURL := p.GetAPIURL()

		providers, err := fetchAuthProviders(cmd.Context(), baseURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching providers: %v\n", err)
			os.Exit(1)
		}
		if len(providers) == 0 {
			fmt.Fprintln(os.Stderr, "Error: no auth providers enabled on server")
			os.Exit(1)
		}

		provider, effectiveMode, err := selectAuthProvider(providers, authMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error selecting auth provider: %v\n", err)
			os.Exit(1)
		}

		var loginData *authLoginData
		switch effectiveMode {
		case authLoginModeDevice:
			loginData, err = oidcLoginViaDeviceFlow(cmd.Context(), baseURL, provider.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "OIDC device login failed: %v\n", err)
				os.Exit(1)
			}
		case authLoginModeBuiltin:
			email, password := resolveBuiltinCredentials()
			loginData, err = builtinLogin(cmd.Context(), baseURL, email, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Built-in login failed: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Error: unsupported login mode %q\n", effectiveMode)
			os.Exit(1)
		}

		cfg.SetCredentials(
			profileName,
			baseURL,
			loginData.AccessToken,
			loginData.RefreshToken,
			loginData.ExpiresAt,
		)
		autoSelectedTeam, autoSelected, autoSelectErr := maybeAutoSelectCurrentTeam(cmd.Context(), cfg, profileName)
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving credentials: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Login successful via provider %q using %s mode (profile: %s)\n", provider.ID, effectiveMode, profileName)
		if autoSelectErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not auto-select current team: %v\n", autoSelectErr)
		}
		if autoSelected {
			fmt.Printf("Auto-selected current team %s (%s)\n", autoSelectedTeam.ID, autoSelectedTeam.Name)
		}
		updatedProfile, err := cfg.GetProfile(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading updated profile: %v\n", err)
			os.Exit(1)
		}
		if shouldShowCurrentTeamSelectionHint(resolveGatewayModeForProfile(cmd.Context(), updatedProfile), updatedProfile.GetCurrentTeamID()) {
			fmt.Println("Global routing requires a locally selected current team for workload commands.")
			fmt.Println("Set it with: s0 team use <team-id>")
			fmt.Println("If you do not have a team yet, create one with: s0 team create --name <name> --home-region <region-id>")
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear local credentials",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := getConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		profileName := cfg.GetActiveProfile()
		p, err := cfg.GetProfile(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading profile: %v\n", err)
			os.Exit(1)
		}

		if token := p.GetToken(); token != "" {
			if err := logoutToken(cmd.Context(), p.GetAPIURL(), token); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: logout API failed: %v\n", err)
			}
		}

		cfg.ClearCredentials(profileName)
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Logged out from profile %q\n", profileName)
	},
}

func resolveBuiltinCredentials() (string, string) {
	email := strings.TrimSpace(authEmail)
	password := authPassword
	reader := bufio.NewReader(os.Stdin)

	if email == "" {
		fmt.Print("Email: ")
		line, _ := reader.ReadString('\n')
		email = strings.TrimSpace(line)
	}
	if password == "" {
		fmt.Print("Password (input hidden): ")
		if disableErr := setTerminalEcho(false); disableErr == nil {
			defer func() {
				_ = setTerminalEcho(true)
				fmt.Println(" [received]")
			}()
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		password = strings.TrimSpace(line)
	}
	return email, password
}

func setTerminalEcho(enabled bool) error {
	mode := "echo"
	if !enabled {
		mode = "-echo"
	}
	cmd := exec.Command("stty", mode)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)

	authLoginCmd.Flags().StringVar(&authEmail, "email", "", "email for built-in provider login")
	authLoginCmd.Flags().StringVar(&authPassword, "password", "", "password for built-in provider login")
	authLoginCmd.Flags().StringVar(&authMode, "mode", string(authLoginModeAuto), "login mode: auto, device, builtin")
	authLoginCmd.MarkFlagsRequiredTogether("email", "password")
}
