package mediagroup

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	mg, err := New(`\\192.168.31.4\buffer\IN\_DONE\testing_sources\Aston_Braun_vs_Sem_Dzhilli_D_PRT260418001456\test.mov`)
	fmt.Println(err)
	fmt.Println(mg)
	fmt.Println(mg.MediaFiles[0])
	cmd, err := mg.MediaFiles[0].ScanAudioRaw()
	if err != nil {
		log.Fatal(err)
	}

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr // прогресс-бар и ошибки пусть идут в stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line) // или парсим
	}
	cmd.Wait()
}
