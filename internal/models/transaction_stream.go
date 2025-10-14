package models

import "time"

// TransactionStreamEvent represents the transaction stream event payload
type TransactionStreamEvent struct {
	Kind                    string                 `json:"kind"`
	TransactionID           string                 `json:"transactionID"`
	ClientID                string                 `json:"clientID"`
	ReferenceNumber         string                 `json:"referenceNumber"`
	ExternalReferenceNumber string                 `json:"externalReferenceNumber"`
	AccountNumber           string                 `json:"accountNumber"`
	OrderTypeCode           string                 `json:"orderTypeCode"`
	TransactionTypeCode     string                 `json:"transactionTypeCode"`
	EntityCode              string                 `json:"entityCode"`
	Amount                  string                 `json:"amount"`
	DetailAmount            DetailAmount           `json:"detailAmount"`
	Currency                string                 `json:"currency"`
	Status                  string                 `json:"status"`
	PaymentType             string                 `json:"paymentType"`
	CreatedAt               time.Time              `json:"createdAt"`
	SuccessfulAt            *time.Time             `json:"successfulAt"`
	InProgressAt            *time.Time             `json:"inProgressAt"`
	ExpiredAt               *time.Time             `json:"expiredAt"`
	CustomerDetails         CustomerDetails        `json:"customerDetails"`
	ItemDetails             []ItemDetail           `json:"itemDetails"`
	PaymentMethod           string                 `json:"paymentMethod"`
	Payment3rdParty         string                 `json:"payment3rdParty"`
	Description             string                 `json:"description"`
	Metadata                map[string]interface{} `json:"metadata"`
	VirtualAccount          *VirtualAccount        `json:"virtualAccount,omitempty"`
	EventType               string                 `json:"eventType"`
}

type DetailAmount struct {
	AdminFee AmountDetailPAPA `json:"adminFee"`
	Net      AmountDetailPAPA `json:"net"`
}

type AmountDetailPAPA struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type CustomerDetails struct {
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	CustomerNumber string `json:"customerNumber"`
}

type ItemDetail struct {
	ID       string `json:"id"`
	Price    string `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type VirtualAccount struct {
	PaymentChannel string      `json:"paymentChannel"`
	BRI            *BRIAccount `json:"bri,omitempty"`
}

type BRIAccount struct {
	VirtualAccountNumber string `json:"virtualAccountNumber"`
}
