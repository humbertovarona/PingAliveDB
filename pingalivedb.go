package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Uso: go run ping.go <IP Address or URL>")
		return
	}

	target := os.Args[1]
	fmt.Println("Destino:", target)

	ip, err := resolveIP(target)
	if err != nil {
		fmt.Println("Error al resolver la IP:", err)
		return
	}

	fmt.Println("Dirección IP:", ip)

	ipType := getIPType(ip)
	fmt.Println("Tipo de IP:", ipType)

	fmt.Println("Realizando ping a:", target)

	rtt, err := ping(ip, ipType)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Ping exitoso. Tiempo de ida y vuelta (RTT): %.2f ms\n", rtt)
	}

	err = createDatabaseIfNotExists()
	if err != nil {
		fmt.Println("Error al crear la base de datos:", err)
		return
	}

	err = savePingResult(ip, target, rtt)
	if err != nil {
		fmt.Println("Error al guardar el resultado del ping en la base de datos:", err)
		return
	}

	fmt.Println("Resultado del ping guardado en la base de datos.")
}

func resolveIP(target string) (string, error) {
	ip := net.ParseIP(target)
	if ip != nil {
		return ip.String(), nil
	}

	addrs, err := net.LookupHost(target)
	if err != nil {
		return "", err
	}

	return addrs[0], nil
}

func getIPType(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP.To4() != nil {
		return "IPv4"
	} else if parsedIP.To16() != nil {
		return "IPv6"
	} else {
		return "Desconocido"
	}
}

func ping(ip, ipType string) (float64, error) {
	timeout := time.Duration(2 * time.Second)
	networkType := ""

	if ipType == "IPv4" {
		networkType = "ip4:icmp"
	} else if ipType == "IPv6" {
		networkType = "ip6:ipv6-icmp"
	} else {
		return 0, fmt.Errorf("Tipo de IP inválido")
	}

	var sum time.Duration
	tries := 3

	for i := 0; i < tries; i++ {
		start := time.Now()

		if ipType == "IPv4" || ipType == "IPv6" {
			conn, err := net.DialTimeout(networkType, ip, timeout)
			if err != nil {
				return 0, err
			}
			conn.Close()
		} else {
			_, err := http.Get("http://" + ip)
			if err != nil {
				return 0, err
			}
		}

		elapsed := time.Since(start)
		sum += elapsed
	}

	averageRTT := float64(sum.Milliseconds()) / float64(tries)
	return averageRTT, nil
}

func createDatabaseIfNotExists() error {
	if _, err := os.Stat("alive.db"); os.IsNotExist(err) {
		db, err := sql.Open("sqlite3", "alive.db")
		if err != nil {
			return err
		}
		defer db.Close()

		query := `
			CREATE TABLE IF NOT EXISTS ping_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				ip TEXT,
				target TEXT,
				rtt REAL,
				timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`

		_, err = db.Exec(query)
		if err != nil {
			return err
		}

		fmt.Println("Base de datos creada: alive.db")
	}

	return nil
}

func savePingResult(ip, target string, rtt float64) error {
	db, err := sql.Open("sqlite3", "alive.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := "INSERT INTO ping_results (ip, target, rtt) VALUES (?, ?, ?);"
	_, err = db.Exec(query, ip, target, rtt)
	if err != nil {
		return err
	}

	return nil
}
