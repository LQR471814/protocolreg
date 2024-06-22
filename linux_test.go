package protocolreg

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip()
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	outputPath := path.Join(cwd, "protocolreg_handler_output")
	os.Remove(outputPath)

	registerId := "protocolreg_test"
	protocol1 := "org.lqr471814.protocolregtest"
	protocol2 := "protocolregtest"

	err = RegisterLinux(registerId, LinuxOptions{
		Metadata: LinuxMetadataOptions{
			Name: "Test protocolreg handler",
		},
		Exec:      fmt.Sprintf(`sh -c "echo '%%u' > %s"`, outputPath),
		Protocols: []string{protocol1, protocol2},
	})
	if err != nil {
		t.Fatal(err)
	}

	nonce := make([]byte, 16)
	_, err = rand.Read(nonce)
	if err != nil {
		t.Fatal(err)
	}

	{
		customUrl := fmt.Sprintf(
			"%s://%s",
			protocol1,
			hex.EncodeToString(nonce),
		)
		err = exec.Command("xdg-open", customUrl).Run()
		if err != nil {
			t.Fatal(err)
		}

		outputFile, err := os.ReadFile(outputPath)
		if os.IsNotExist(err) {
			t.Fatalf("protocolreg test handler did not receive custom url request")
		}
		if err != nil {
			t.Fatal(err)
		}

		require.Contains(t, string(outputFile), customUrl)
	}

	{
		customUrl := fmt.Sprintf(
			"%s://%s",
			protocol2,
			hex.EncodeToString(nonce),
		)
		err = exec.Command("xdg-open", customUrl).Run()
		if err != nil {
			t.Fatal(err)
		}

		outputFile, err := os.ReadFile(outputPath)
		if os.IsNotExist(err) {
			t.Fatalf("protocolreg test handler did not receive custom url request")
		}
		if err != nil {
			t.Fatal(err)
		}

		require.Contains(t, string(outputFile), customUrl)
	}

	{
		err = UnregisterLinux(registerId)
		if err != nil {
			t.Fatal(err)
		}

		err = exec.Command("xdg-open", fmt.Sprintf("%s://something", protocol1)).Run()
		require.NotNil(t, err)

		err = exec.Command("xdg-open", fmt.Sprintf("%s://something", protocol2)).Run()
		require.NotNil(t, err)
	}
}
