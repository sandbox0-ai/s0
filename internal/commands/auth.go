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
	authEmail      string
	authPassword   string
	authHomeRegion string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Authenticate s0 CLI using OIDC or built-in account login.`,
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

		provider := providers[0]
		var loginData *authLoginData
		switch provider.Type {
		case "oidc":
			loginData, err = oidcLoginViaBrowser(cmd.Context(), baseURL, provider.ID, authHomeRegion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "OIDC login failed: %v\n", err)
				os.Exit(1)
			}
		case "builtin":
			email, password := resolveBuiltinCredentials()
			loginData, err = builtinLogin(cmd.Context(), baseURL, email, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Built-in login failed: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Error: unsupported provider type %q\n", provider.Type)
			os.Exit(1)
		}

		cfg.SetCredentials(
			profileName,
			baseURL,
			loginData.AccessToken,
			loginData.RefreshToken,
			loginData.ExpiresAt,
			toRegionalSessionConfig(loginData),
		)
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving credentials: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Login successful via provider %q (profile: %s)\n", provider.ID, profileName)
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
	authLoginCmd.Flags().StringVar(&authHomeRegion, "home-region", "", "home region ID for first-time OIDC provisioning in global mode")
	authLoginCmd.MarkFlagsRequiredTogether("email", "password")
}
