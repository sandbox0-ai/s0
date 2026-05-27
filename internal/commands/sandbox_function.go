package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	functionSandboxID  string
	functionMethod     string
	functionPath       string
	functionHandler    string
	functionHeader     []string
	functionQuery      []string
	functionBodyStdin  bool
	functionBodyData   string
	functionBodyBase64 string
	functionTimeoutMS  int32
)

// sandboxFunctionCmd represents the sandbox function command group.
var sandboxFunctionCmd = &cobra.Command{
	Use:   "function",
	Short: "Invoke sandbox functions",
	Long:  `Invoke functions stored under /workspace/functions inside a sandbox.`,
}

// sandboxFunctionInvokeCmd invokes a sandbox function.
var sandboxFunctionInvokeCmd = &cobra.Command{
	Use:   "invoke <name>",
	Short: "Invoke a sandbox function",
	Long:  `Invoke /workspace/functions/<name>.py with the Python function runtime.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req, err := buildFunctionInvokeRequest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building function request: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.Sandbox(functionSandboxID).InvokeFunction(cmd.Context(), args[0], req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error invoking function: %v\n", err)
			os.Exit(1)
		}

		if cfgFormat == "json" || cfgFormat == "yaml" {
			if err := getFormatter().Format(os.Stdout, resp); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			if resp.Status >= 400 {
				os.Exit(1)
			}
			return
		}

		if bodyBase64, ok := resp.BodyBase64.Get(); ok && bodyBase64 != "" {
			body, err := base64.StdEncoding.DecodeString(bodyBase64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error decoding function response body: %v\n", err)
				os.Exit(1)
			}
			if _, err := os.Stdout.Write(body); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Function returned status %d\n", resp.Status)
		}
		if resp.Status >= 400 {
			os.Exit(1)
		}
	},
}

func buildFunctionInvokeRequest() (apispec.FunctionInvokeRequest, error) {
	req := apispec.FunctionInvokeRequest{}
	if functionMethod != "" {
		req.Method = apispec.NewOptString(functionMethod)
	}
	if functionPath != "" {
		req.Path = apispec.NewOptString(functionPath)
	}
	if functionHandler != "" {
		req.Handler = apispec.NewOptString(functionHandler)
	}
	if functionTimeoutMS < 0 {
		return req, fmt.Errorf("--timeout-ms must be >= 0")
	}
	if functionTimeoutMS > 0 {
		req.TimeoutMs = apispec.NewOptInt32(functionTimeoutMS)
	}
	if len(functionHeader) > 0 {
		headers, err := parseFunctionFields(functionHeader)
		if err != nil {
			return req, err
		}
		req.Headers = apispec.NewOptFunctionInvokeRequestHeaders(apispec.FunctionInvokeRequestHeaders(headers))
	}
	if len(functionQuery) > 0 {
		query, err := parseFunctionFields(functionQuery)
		if err != nil {
			return req, err
		}
		req.Query = apispec.NewOptFunctionInvokeRequestQuery(apispec.FunctionInvokeRequestQuery(query))
	}
	bodyBase64, err := buildFunctionBodyBase64()
	if err != nil {
		return req, err
	}
	if bodyBase64 != "" {
		req.BodyBase64 = apispec.NewOptString(bodyBase64)
	}
	return req, nil
}

func buildFunctionBodyBase64() (string, error) {
	if functionBodyBase64 != "" {
		if functionBodyStdin || functionBodyData != "" {
			return "", fmt.Errorf("--body-base64 cannot be combined with --stdin or --body")
		}
		if _, err := base64.StdEncoding.DecodeString(functionBodyBase64); err != nil {
			return "", fmt.Errorf("--body-base64 must be valid base64: %w", err)
		}
		return functionBodyBase64, nil
	}
	if !functionBodyStdin && functionBodyData == "" {
		return "", nil
	}
	body, err := readCommandContent(functionBodyStdin, functionBodyData, os.Stdin)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(body), nil
}

func parseFunctionFields(values []string) (map[string][]string, error) {
	out := make(map[string][]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid field %q, expected KEY=VALUE", value)
		}
		out[key] = append(out[key], val)
	}
	return out, nil
}

func init() {
	sandboxFunctionCmd.AddCommand(sandboxFunctionInvokeCmd)

	sandboxFunctionCmd.PersistentFlags().StringVarP(&functionSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxFunctionCmd.MarkPersistentFlagRequired("sandbox-id")

	sandboxFunctionInvokeCmd.Flags().StringVar(&functionMethod, "method", "", "logical request method")
	sandboxFunctionInvokeCmd.Flags().StringVar(&functionPath, "path", "", "logical request path")
	sandboxFunctionInvokeCmd.Flags().StringVar(&functionHandler, "handler", "", "handler name inside the Python module")
	sandboxFunctionInvokeCmd.Flags().StringArrayVar(&functionHeader, "header", nil, "request header (KEY=VALUE, can be repeated)")
	sandboxFunctionInvokeCmd.Flags().StringArrayVar(&functionQuery, "query", nil, "query parameter (KEY=VALUE, can be repeated)")
	sandboxFunctionInvokeCmd.Flags().BoolVar(&functionBodyStdin, "stdin", false, "read request body from stdin")
	sandboxFunctionInvokeCmd.Flags().StringVar(&functionBodyData, "body", "", "request body")
	sandboxFunctionInvokeCmd.Flags().StringVar(&functionBodyBase64, "body-base64", "", "base64-encoded request body")
	sandboxFunctionInvokeCmd.Flags().Int32Var(&functionTimeoutMS, "timeout-ms", 0, "per-invocation timeout in milliseconds")

	sandboxCmd.AddCommand(sandboxFunctionCmd)
}
