package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GatewayClient calls connect-gateway with a user's JWT token.
type GatewayClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewGatewayClient(baseURL string) *GatewayClient {
	return &GatewayClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// BalanceResponse from wallet balance endpoint.
type BalanceResponse struct {
	Balance  json.Number `json:"balance"`
	Currency string      `json:"currency"`
}

// BookingSummary is a minimal booking for display.
type BookingSummary struct {
	ID          string `json:"id"`
	Reference   string `json:"reference"`
	Status      string `json:"status"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Date        string `json:"date"`
	Passengers  int    `json:"passengers"`
}

type BookingsResponse struct {
	Bookings []BookingSummary `json:"bookings"`
}

type Currency struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	ExchangeRate float64 `json:"exchangeRate"`
}

type CurrenciesResponse struct {
	Currencies []Currency `json:"currencies"`
}

// AdminDepositRequest for admin wallet deposit.
type AdminDepositRequest struct {
	UserID   string  `json:"userId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Note     string  `json:"note"`
}

type TransactionResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transactionId"`
	Message       string `json:"message"`
}

type UpdateCurrencyRequest struct {
	ExchangeRate float64 `json:"exchangeRate"`
}

// GetWalletBalance fetches wallet balance for a currency.
func (c *GatewayClient) GetWalletBalance(token, currency string) (*BalanceResponse, error) {
	path := fmt.Sprintf("/api/v1/wallet/balance/%s", currency)
	data, err := c.doGet(token, path)
	if err != nil {
		return nil, err
	}
	var resp BalanceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode balance: %w", err)
	}
	return &resp, nil
}

// GetUserBookings fetches the user's bookings.
func (c *GatewayClient) GetUserBookings(token string) (*BookingsResponse, error) {
	data, err := c.doGet(token, "/api/v1/bookings")
	if err != nil {
		return nil, err
	}
	var resp BookingsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode bookings: %w", err)
	}
	return &resp, nil
}

// GetCurrencyList fetches available currencies.
func (c *GatewayClient) GetCurrencyList(token string) (*CurrenciesResponse, error) {
	data, err := c.doGet(token, "/api/v1/currencies")
	if err != nil {
		return nil, err
	}
	var resp CurrenciesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}
	return &resp, nil
}

// AdminDeposit creates a wallet deposit.
func (c *GatewayClient) AdminDeposit(token string, req AdminDepositRequest) (*TransactionResponse, error) {
	data, err := c.doPost(token, "/api/v1/admin/wallets/deposit", req)
	if err != nil {
		return nil, err
	}
	var resp TransactionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode deposit: %w", err)
	}
	return &resp, nil
}

// AdminUpdateCurrency updates a currency exchange rate.
func (c *GatewayClient) AdminUpdateCurrency(token, code string, req UpdateCurrencyRequest) error {
	path := fmt.Sprintf("/api/v1/admin/currencies/%s", code)
	_, err := c.doPut(token, path, req)
	return err
}

// DoRequest performs a generic gateway request and returns status + body.
func (c *GatewayClient) DoRequest(method, token, path string, body interface{}) (int, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, nil
}

func (c *GatewayClient) doGet(token, path string) ([]byte, error) {
	status, body, err := c.DoRequest(http.MethodGet, token, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, &GatewayError{StatusCode: status, Body: string(body)}
	}
	return body, nil
}

func (c *GatewayClient) doPost(token, path string, payload interface{}) ([]byte, error) {
	status, body, err := c.DoRequest(http.MethodPost, token, path, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, &GatewayError{StatusCode: status, Body: string(body)}
	}
	return body, nil
}

func (c *GatewayClient) doPut(token, path string, payload interface{}) ([]byte, error) {
	status, body, err := c.DoRequest(http.MethodPut, token, path, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, &GatewayError{StatusCode: status, Body: string(body)}
	}
	return body, nil
}

// TicketSummary is a minimal ticket for display.
type TicketSummary struct {
	ID           string `json:"id"`
	TicketNumber string `json:"ticketNumber"`
	Category     string `json:"category"`
	Status       string `json:"status"`
	Priority     string `json:"priority"`
}

type ListTicketsResponse struct {
	Tickets []TicketSummary `json:"tickets"`
}

// ListAgentTickets fetches the agent's assigned tickets via ConnectRPC.
func (c *GatewayClient) ListAgentTickets(token string) ([]TicketSummary, error) {
	body := map[string]interface{}{
		"statusFilter": "open",
	}
	data, err := c.doConnectPost(token, "/eticket.support.v1.SupportService/ListTickets", body)
	if err != nil {
		return nil, err
	}
	var resp ListTicketsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode tickets: %w", err)
	}
	return resp.Tickets, nil
}

// SendAgentMessage sends an agent reply to a ticket via ConnectRPC.
func (c *GatewayClient) SendAgentMessage(token, ticketID, content string) error {
	body := map[string]interface{}{
		"ticketId": ticketID,
		"content":  content,
	}
	_, err := c.doConnectPost(token, "/eticket.support.v1.SupportService/SendAgentMessage", body)
	return err
}

// ticketStatusEnum maps short status names to proto enum names.
var ticketStatusEnum = map[string]string{
	"new":                 "TICKET_STATUS_NEW",
	"open":                "TICKET_STATUS_OPEN",
	"pending":             "TICKET_STATUS_PENDING",
	"waiting_on_customer": "TICKET_STATUS_WAITING_ON_CUSTOMER",
	"escalated":           "TICKET_STATUS_ESCALATED",
	"resolved":            "TICKET_STATUS_RESOLVED",
	"closed":              "TICKET_STATUS_CLOSED",
}

// UpdateTicketStatus updates a ticket's status via ConnectRPC.
func (c *GatewayClient) UpdateTicketStatus(token, ticketID, status string) error {
	protoStatus, ok := ticketStatusEnum[status]
	if !ok {
		return fmt.Errorf("unknown status: %s", status)
	}
	body := map[string]interface{}{
		"ticketId": ticketID,
		"status":   protoStatus,
	}
	_, err := c.doConnectPost(token, "/eticket.support.v1.SupportService/UpdateTicketStatus", body)
	return err
}

// AssignTicket assigns a ticket to the calling agent via ConnectRPC.
func (c *GatewayClient) AssignTicket(token, ticketID string) error {
	body := map[string]interface{}{
		"ticketId": ticketID,
	}
	_, err := c.doConnectPost(token, "/eticket.support.v1.SupportService/AssignTicket", body)
	return err
}

// doConnectPost performs a ConnectRPC POST with JSON encoding.
func (c *GatewayClient) doConnectPost(token, path string, body interface{}) ([]byte, error) {
	status, respBody, err := c.DoRequest(http.MethodPost, token, path, body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, &GatewayError{StatusCode: status, Body: string(respBody)}
	}
	return respBody, nil
}

// AIRequest is the request body for ProcessNaturalLanguageRequest.
type AIRequest struct {
	UserID         string            `json:"userId"`
	PartnerID      string            `json:"partnerId"`
	Message        string            `json:"message"`
	ConversationID string            `json:"conversationId,omitempty"`
	Language       string            `json:"language"`
	UserContext    map[string]string `json:"userContext,omitempty"`
	Channel        string            `json:"channel,omitempty"`
}

// AIResponse is the response from ProcessNaturalLanguageRequest.
type AIResponse struct {
	Success         bool      `json:"success"`
	ErrorMessage    string    `json:"errorMessage"`
	ConversationID  string    `json:"conversationId"`
	ResponseMessage string    `json:"responseMessage"`
	DetectedIntent  string    `json:"detectedIntent"`
	Suggestions     []string  `json:"suggestions"`
	Action          *AIAction `json:"action,omitempty"`
}

// AIAction is the parsed action from AI.
type AIAction struct {
	Action     string            `json:"action"`
	Parameters map[string]string `json:"parameters"`
	Valid      bool              `json:"valid"`
	Confidence float64           `json:"confidence"`
}

// AskAI calls ProcessNaturalLanguageRequest via ConnectRPC.
func (c *GatewayClient) AskAI(token string, req AIRequest) (*AIResponse, error) {
	data, err := c.doConnectPost(token, "/eticket.ai.v1.AIService/ProcessNaturalLanguageRequest", req)
	if err != nil {
		return nil, err
	}
	var resp AIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode AI response: %w", err)
	}
	return &resp, nil
}

// GatewayError represents an HTTP error from the gateway.
type GatewayError struct {
	StatusCode int
	Body       string
}

func (e *GatewayError) Error() string {
	return fmt.Sprintf("gateway returned %d: %s", e.StatusCode, e.Body)
}
