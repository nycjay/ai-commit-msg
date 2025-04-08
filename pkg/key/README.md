# Key Management Package

This package provides secure storage and retrieval of API keys across different platforms.

## Features

- **Cross-Platform Support**:
  - macOS: Uses the Keychain via the `security` command-line tool
  - Windows: Uses the Windows Credential Manager via the `wincred` package
  - Other platforms: Can use environment variables as fallback

- **Secure Storage**: API keys are stored securely in the platform's credential store, not in plaintext configuration files

- **Validation**: Basic API key format validation to catch common errors

## Usage

```go
// Create a new key manager
km := key.NewKeyManager(verbose)

// Check if credential store is available
if km.CredentialStoreAvailable() {
    // Store an API key
    err := km.StoreInKeychain("sk_ant_your_api_key")
    if err != nil {
        // Handle error
    }
    
    // Retrieve an API key
    apiKey, err := km.GetFromKeychain()
    if err != nil {
        // Handle error
    }
}

// Get API key with automatic fallback:
// 1. Try command-line flag (if provided)
// 2. Try environment variable
// 3. Try credential store
apiKey, err := km.GetKey(cmdLineKey)
if err != nil {
    // No API key found in any source
}
```

## Platform-Specific Details

### macOS

On macOS, the package uses the `security` command-line tool to interact with the Keychain:

- **Service Name**: `ai-commit-msg`
- **Account Name**: `anthropic-api-key`

### Windows

On Windows, the package uses the Windows Credential Manager via the `github.com/danieljoos/wincred` package:

- **Target Name**: `ai-commit-msg:anthropic-api-key`
- **Username**: `anthropic-api-key`
- **Persistence**: `LocalMachine` (available to all users on the machine)

### Other Platforms

On other platforms, the package will fall back to environment variables. The following environment variable is checked:

- `ANTHROPIC_API_KEY`

## Security

API keys are sensitive credentials and should be protected. This package helps by:

1. Never storing the API key in plaintext configuration files
2. Using the platform's secure credential storage
3. Limiting access to the stored credentials

## Debugging

When verbose mode is enabled, the key manager will log information about its operations. This can be useful for diagnosing issues with credential storage.
