package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)


// AWS Secrets Managerから認証情報を取得する関数
func getCredentialsFromSecretsManager(ctx context.Context, secretName string) ([]byte, error) {
	// IAMロールからAWSの認証情報を自動で読み込む
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("ap-northeast-1")) // ← 東京リージョン
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	// Secrets Managerのクライアントを作成
	svc := secretsmanager.NewFromConfig(cfg)

	// シークレットを取得
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("unable to get secret value, %v", err)
	}

	// シークレットは文字列なので、バイト配列に変換して返す
	return []byte(*result.SecretString), nil
}


func main() {
	// 背景情報としてcontextを生成する
	ctx := context.Background()

	secretName := "discord-bot/gcp-credentials"

	// ファイルを読む代わりに、Secrets Managerから認証情報を取得する
	b, err := getCredentialsFromSecretsManager(ctx, secretName)
	if err != nil {
		log.Fatalf("Failed to get credentials from Secrets Manager: %v", err)
	}

	// 環境変数 "SPREADSHEET_ID" からIDを読み込む
	spreadsheetId := os.Getenv("SPREADSHEET_ID")
	if spreadsheetId == "" {
		log.Fatalf("環境変数 SPREADSHEET_ID が設定されていません。")
	}

	// 取得したいシート名とセル範囲を指定する
	readRange := "シート1!A:C"

	// 認証情報を作成する
	config, err := google.JWTConfigFromJSON(b, sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// 認証済みクライアントを生成する
	client := config.Client(ctx)

	// 認証済みクライアントを使って、新しいSheetsサービスを生成する
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// データを取得する
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
		return
	}

	// CSVファイルを作成する
	file, err := os.Create("serif-list.csv")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 取得したデータをCSVに書き込む
	for _, row := range resp.Values {
		var stringRow []string
		for _, cell := range row {
			stringRow = append(stringRow, fmt.Sprintf("%v", cell))
		}
		if err := writer.Write(stringRow); err != nil {
			log.Fatalf("Failed to write to file: %v", err)
		}
	}

	fmt.Println("CSV file 'serif-list.csv' created successfully.")
}