//go:build !linux && !darwin

package secret

func readSecretNoEcho(prompt string) (string, error) {
	return "", errNotSupported
}
