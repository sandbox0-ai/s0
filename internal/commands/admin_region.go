package commands

import (
	"fmt"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

type adminRegionOptions struct {
	id                 string
	displayName        string
	regionalGatewayURL string
	meteringExportURL  string
	enabled            bool
}

func (o *adminRegionOptions) addCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.id, "id", "", "region ID (required)")
	cmd.Flags().StringVar(&o.displayName, "display-name", "", "region display name")
	cmd.Flags().StringVar(&o.regionalGatewayURL, "regional-gateway-url", "", "regional gateway URL (required)")
	cmd.Flags().StringVar(&o.regionalGatewayURL, "edge-gateway-url", "", "alias for --regional-gateway-url")
	cmd.Flags().StringVar(&o.meteringExportURL, "metering-export-url", "", "metering export URL")
	cmd.Flags().BoolVar(&o.enabled, "enabled", true, "whether the region is enabled")
}

func (o *adminRegionOptions) addUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.displayName, "display-name", "", "region display name")
	cmd.Flags().StringVar(&o.regionalGatewayURL, "regional-gateway-url", "", "regional gateway URL")
	cmd.Flags().StringVar(&o.regionalGatewayURL, "edge-gateway-url", "", "alias for --regional-gateway-url")
	cmd.Flags().StringVar(&o.meteringExportURL, "metering-export-url", "", "metering export URL")
	cmd.Flags().BoolVar(&o.enabled, "enabled", false, "whether the region is enabled")
}

func newAdminRegionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "region",
		Short: "Manage Global Gateway regions",
		Long:  `List and manage Global Gateway region directory entries. These commands require system-admin access.`,
	}

	cmd.AddCommand(newAdminRegionListCommand())
	cmd.AddCommand(newAdminRegionGetCommand())
	cmd.AddCommand(newAdminRegionCreateCommand())
	cmd.AddCommand(newAdminRegionUpdateCommand())
	cmd.AddCommand(newAdminRegionDeleteCommand())
	return cmd
}

func newAdminRegionListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Global Gateway regions",
		Long:  `List region directory entries from the Global Gateway. This is a platform admin operation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			res, err := client.API().RegionsGet(cmd.Context())
			if err != nil {
				return fmt.Errorf("list regions: %w", err)
			}

			successRes, ok := res.(*apispec.SuccessRegionListResponse)
			if !ok {
				return fmt.Errorf("list regions: unexpected response type %T", res)
			}

			data, ok := successRes.Data.Get()
			if !ok {
				return fmt.Errorf("list regions: missing response data")
			}

			return getFormatter().Format(cmd.OutOrStdout(), data.Regions)
		},
	}
}

func newAdminRegionGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <region-id>",
		Short: "Get Global Gateway region details",
		Long:  `Get one region directory entry from the Global Gateway. This is a platform admin operation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			res, err := client.API().RegionsIDGet(cmd.Context(), apispec.RegionsIDGetParams{ID: args[0]})
			if err != nil {
				return fmt.Errorf("get region: %w", err)
			}

			successRes, ok := res.(*apispec.SuccessRegionResponse)
			if !ok {
				return fmt.Errorf("get region: unexpected response type %T", res)
			}

			data, ok := successRes.Data.Get()
			if !ok {
				return fmt.Errorf("get region: missing response data")
			}

			return getFormatter().Format(cmd.OutOrStdout(), &data)
		},
	}
}

func newAdminRegionCreateCommand() *cobra.Command {
	opts := &adminRegionOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Global Gateway region",
		Long:  `Create a new region directory entry in the Global Gateway. This is a platform admin operation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildCreateRegionRequest(cmd, *opts)
			if err != nil {
				return err
			}

			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			res, err := client.API().RegionsPost(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create region: %w", err)
			}

			successRes, ok := res.(*apispec.SuccessRegionResponse)
			if !ok {
				return fmt.Errorf("create region: unexpected response type %T", res)
			}

			data, ok := successRes.Data.Get()
			if !ok {
				return fmt.Errorf("create region: missing response data")
			}

			return getFormatter().Format(cmd.OutOrStdout(), &data)
		},
	}
	opts.addCreateFlags(cmd)
	return cmd
}

func newAdminRegionUpdateCommand() *cobra.Command {
	opts := &adminRegionOptions{}
	cmd := &cobra.Command{
		Use:   "update <region-id>",
		Short: "Update a Global Gateway region",
		Long:  `Update an existing region directory entry in the Global Gateway. This is a platform admin operation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildUpdateRegionRequest(cmd, *opts)
			if err != nil {
				return err
			}

			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			res, err := client.API().RegionsIDPut(cmd.Context(), req, apispec.RegionsIDPutParams{ID: args[0]})
			if err != nil {
				return fmt.Errorf("update region: %w", err)
			}

			successRes, ok := res.(*apispec.SuccessRegionResponse)
			if !ok {
				return fmt.Errorf("update region: unexpected response type %T", res)
			}

			data, ok := successRes.Data.Get()
			if !ok {
				return fmt.Errorf("update region: missing response data")
			}

			return getFormatter().Format(cmd.OutOrStdout(), &data)
		},
	}
	opts.addUpdateFlags(cmd)
	return cmd
}

func newAdminRegionDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <region-id>",
		Short: "Delete a Global Gateway region",
		Long:  `Delete a region directory entry from the Global Gateway. This is a platform admin operation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			res, err := client.API().RegionsIDDelete(cmd.Context(), apispec.RegionsIDDeleteParams{ID: args[0]})
			if err != nil {
				return fmt.Errorf("delete region: %w", err)
			}

			successRes, ok := res.(*apispec.SuccessMessageResponse)
			if !ok {
				return fmt.Errorf("delete region: unexpected response type %T", res)
			}

			if data, ok := successRes.Data.Get(); ok {
				if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), message)
					return err
				}
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Region %s deleted successfully\n", args[0])
			return err
		},
	}
}

func buildCreateRegionRequest(cmd *cobra.Command, opts adminRegionOptions) (*apispec.CreateRegionRequest, error) {
	regionID := strings.TrimSpace(opts.id)
	if regionID == "" {
		return nil, fmt.Errorf("--id is required")
	}

	regionalGatewayURL, err := trimRegionalGatewayURL(cmd, opts.regionalGatewayURL, true)
	if err != nil {
		return nil, err
	}

	req := &apispec.CreateRegionRequest{
		ID:                 regionID,
		RegionalGatewayURL: regionalGatewayURL,
	}
	if displayName := strings.TrimSpace(opts.displayName); displayName != "" {
		req.DisplayName = apispec.NewOptString(displayName)
	}
	if cmd.Flags().Changed("metering-export-url") {
		req.MeteringExportURL = apispec.NewOptNilString(strings.TrimSpace(opts.meteringExportURL))
	}
	if cmd.Flags().Changed("enabled") {
		req.Enabled = apispec.NewOptBool(opts.enabled)
	}
	return req, nil
}

func buildUpdateRegionRequest(cmd *cobra.Command, opts adminRegionOptions) (*apispec.UpdateRegionRequest, error) {
	req := &apispec.UpdateRegionRequest{}
	hasChange := false

	if cmd.Flags().Changed("display-name") {
		if displayName := strings.TrimSpace(opts.displayName); displayName != "" {
			req.DisplayName = apispec.NewOptString(displayName)
			hasChange = true
		}
	}

	if flagChanged(cmd, "regional-gateway-url", "edge-gateway-url") {
		regionalGatewayURL, err := trimRegionalGatewayURL(cmd, opts.regionalGatewayURL, false)
		if err != nil {
			return nil, err
		}
		req.RegionalGatewayURL = apispec.NewOptString(regionalGatewayURL)
		hasChange = true
	}

	if cmd.Flags().Changed("metering-export-url") {
		req.MeteringExportURL = apispec.NewOptNilString(strings.TrimSpace(opts.meteringExportURL))
		hasChange = true
	}

	if cmd.Flags().Changed("enabled") {
		req.Enabled = apispec.NewOptBool(opts.enabled)
		hasChange = true
	}

	if !hasChange {
		return nil, fmt.Errorf("at least one field must be set for update")
	}
	return req, nil
}

func trimRegionalGatewayURL(cmd *cobra.Command, value string, required bool) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed, nil
	}
	if required || flagChanged(cmd, "regional-gateway-url", "edge-gateway-url") {
		return "", fmt.Errorf("--regional-gateway-url is required")
	}
	return "", nil
}

func flagChanged(cmd *cobra.Command, names ...string) bool {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}
