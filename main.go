package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Функция для загрузки товаров на Озон
func loadProductsToOzon(filename string, apiKey string) {
	// Чтение содержимого CSV-файла
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Парсинг CSV-данных
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV data: %v", err)
	}

	// Создание канала для ограничения количества горутин
	semaphore := make(chan struct{}, 5) // Максимальное количество горутин

	var wg sync.WaitGroup

	// Обработка каждой записи из CSV-файла
	for _, record := range records {
		wg.Add(1)
		go func(record []string) {
			defer wg.Done()

			// Ожидание места в канале
			semaphore <- struct{}{}

			// Извлечение данных из записи CSV
			productName := record[0]
			productPrice := parsePrice(record[1])

			// Формирование запроса на API Озона
			url := fmt.Sprintf("https://api-seller.ozon.ru/v1/product/import?sku=%s&price=%f&apikey=%s", productName, productPrice, apiKey)
			resp, err := http.Post(url, "application/json", bytes.NewReader(nil))
			if err != nil {
				log.Printf("API request failed: %v", err)
				<-semaphore // Освобождение места в канале при ошибке
				return
			}
			defer resp.Body.Close()

			// Проверка кода ответа
			if resp.StatusCode != http.StatusOK {
				log.Printf("API request failed with status code %d", resp.StatusCode)
				<-semaphore // Освобождение места в канале при ошибке
				return
			}

			// Обработка успешного ответа
			log.Printf("Product '%s' successfully imported", productName)

			<-semaphore // Освобождение места в канале после успешной загрузки товара
		}(record)
	}

	// Ожидание завершения всех горутин
	wg.Wait()

	log.Println("All products imported")
}

// Функция для преобразования строки с ценой в число
func parsePrice(priceStr string) float64 {
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		log.Printf("Error parsing price '%s': %v", priceStr, err)
		return 0.0
	}
	return price
}
func main() {
	// Параметры загрузки товаров
	filename := "products.csv"
	apiKey := "c6ac6335-ee09-42ae-a0c1-1d513cb437b2"
	loadProductsToOzon(filename, apiKey)
}
