package context

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func BuildPDF(workDIR, texFN, pdfFN string) error {
	log.Printf("generating PDF -> %s\n", pdfFN)

	x := exec.Command("context", "--result", pdfFN, texFN)
	x.Stderr = os.Stderr
	x.Stdout = os.Stdout
	x.Dir = workDIR
	err := x.Run()
	if err != nil {
		return fmt.Errorf("ConTEXt error: %w", err)
	}

	_, err = os.Stat(pdfFN)
	if os.IsNotExist(err) {
		return fmt.Errorf("missing ConTEXt output: %w", err)
	} else if err != nil {
		return fmt.Errorf("missing ConTEXt output: %w", err)
	}
	return nil
}
