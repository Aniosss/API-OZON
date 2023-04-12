package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Product структура для представления товара
type Product struct {
	Name        string
	Description string
	Price       float64
}

func main() {
	// Чтение CSV-файла с товарами
	file, err := os.Open("products.csv")
	if err != nil {
		log.Fatal("Failed to open CSV file:", err)
	}
	defer file.Close()

	// Создание CSV-ридера
	reader := csv.NewReader(file)

	// Чтение всех записей из CSV-файла
	products := make([]Product, 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Failed to read CSV record:", err)
		}

		price, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			log.Fatal(err)
		}

		// Создание структуры Product из записи CSV
		product := Product{
			Name:        record[0],
			Description: record[1],
			Price:       price,
		}
		products = append(products, product)
	}

	// Создание клиента HTTP
	client := &http.Client{}

	// Создание канала для ограничения количества одновременно выполняющихся горутин
	concurrentGoroutines := make(chan struct{}, 10) // ограничение на 10 горутин

	// Создание WaitGroup для ожидания завершения всех горутин
	wg := sync.WaitGroup{}

	// Импорт товаров в Озон
	for _, product := range products {
		// Добавление горутины в WaitGroup
		wg.Add(1)

		// Отправка запроса на импорт товара в отдельной  горутине
		go func(p Product) {
			// Ожидание доступа в канал
			concurrentGoroutines <- struct{}{}

			// Отправка запроса на API Озона
			url := "https://api.ozon.ru/composer-api.bx/page/json/v1"
			reqBody := fmt.Sprintf(`{
				"name": "%s",
				"description": "%s",
				"price": %.2f
			}`, p.Name, p.Description, p.Price)
			resp, err := client.Post(url, "application/json", strings.NewReader(reqBody))
			if err != nil {
				log.Println("Failed to send request:", err)
				wg.Done()
				<-concurrentGoroutines
				return
			}
			defer resp.Body.Close()

			// Проверка статуса ответа
			if resp.StatusCode != http.StatusOK {
				log.Println("Failed to import product:", resp.Status)
				wg.Done()
				<-concurrentGoroutines
				return
			}

			// Логирование успешного импорта товара
			log.Println("Product imported successfully:", p.Name)

			// Операция выполнена, освобождение канала и завершение горутины
			wg.Done()
			<-concurrentGoroutines
		}(product)
	}

	// Ожидание завершения всех горутин
	wg.Wait()

	log.Println("All products imported successfully!")

}
