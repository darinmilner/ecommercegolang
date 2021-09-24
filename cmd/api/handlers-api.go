package main

import (
	"ecommerce/internal/cards"
	"ecommerce/internal/encryption"
	"ecommerce/internal/models"
	"ecommerce/internal/urlsigner"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v72"
	"golang.org/x/crypto/bcrypt"
)

type stripePayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"paymentMethod"`
	Email         string `json:"email"`
	CardBrand     string `json:"cardBrand"`
	ExpiryMonth   int    `json:"expiryMonth"`
	ExpiryYear    int    `json:"expiryYear"`
	LastFour      string `json:"lastFour"`
	Plan          string `json:"plan"`
	ProductID     string `json:"productId"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
}

type jsonResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      int    `json:"id,omitempty"`
}

//GetPaymentIntent gets the info for a payment intent
func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var payload stripePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secretKey,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	okay := true

	pi, msg, err := card.ChargeCard(payload.Currency, amount)

	if err != nil {
		okay = false
	}

	if okay {
		out, err := json.MarshalIndent(pi, "", "    ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	} else {
		j := jsonResponse{
			OK:      false,
			Message: msg,
			Content: "",
		}

		out, err := json.MarshalIndent(j, "", "   ")

		if err != nil {
			app.errorLog.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}

}

//GetWidgetByID gets the widget by its ID
func (app *application) GetWidgetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	widgetID, err := strconv.Atoi(id)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	widget, err := app.DB.GetWidget(widgetID)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	out, err := json.MarshalIndent(widget, "", "   ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

//CreateCustomerAndSubscribe creates a new customer and subscribes them to a subscription
func (app *application) CreateCustomerAndSubscribe(w http.ResponseWriter, r *http.Request) {
	var data stripePayload

	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Println(data.Email, data.LastFour, data.PaymentMethod, data.Plan)

	card := cards.Card{
		Secret:   app.config.stripe.secretKey,
		Key:      app.config.stripe.key,
		Currency: data.Currency,
	}

	okay := true
	txnMsg := "Transaction successful"
	var subscription *stripe.Subscription

	stripeCustomer, msg, err := card.CreateCustomer(data.PaymentMethod, data.Email)
	if err != nil {
		app.errorLog.Println(err)
		okay = false
		txnMsg = msg
	}

	if okay {
		subscription, err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email, data.LastFour, "")
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			txnMsg = "Error subscribing customer"
		}

		app.infoLog.Println("Subscription ID ", subscription.ID)
	}

	if okay {
		//create customer
		productID, err := strconv.Atoi(data.ProductID)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			return
		}

		customerID, err := app.SaveCustomer(data.FirstName, data.LastName, data.Email)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			return
		}

		//create transaction
		amount, err := strconv.Atoi(data.Amount)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			return
		}
		// expiryMonth, err := strconv.Atoi(data.ExpiryMonth)
		// if err != nil {
		// 	app.errorLog.Println(err)
		// 	okay = false
		// 	return
		// }
		// expiryYear, err := strconv.Atoi(data.ExpiryYear)
		// if err != nil {
		// 	app.errorLog.Println(err)
		// 	okay = false
		// 	return
		// }

		txn := models.Transaction{
			Amount:              amount,
			Currency:            "usd",
			LastFour:            data.LastFour,
			ExpiryMonth:         data.ExpiryMonth,
			ExpiryYear:          data.ExpiryYear,
			TransactionStatusID: 2,
		}

		txnID, err := app.SaveTransaction(txn)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			return
		}

		//create Order
		order := models.Order{
			WidgetID:      productID,
			TransactionID: txnID,
			CustomerID:    customerID,
			StatusID:      1,
			Quantity:      1,
			Amount:        amount,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		_, err = app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			return
		}

	}

	resp := jsonResponse{
		OK:      okay,
		Message: txnMsg,
	}

	out, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

//SaveCustomer saves a customer to the DB
func (app *application) SaveCustomer(firstName, lastName, email string) (int, error) {
	customer := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	id, err := app.DB.InsertCustomer(customer)
	if err != nil {
		app.errorLog.Println(err)
		return 0, err
	}

	return id, nil
}

//SaveTransaction saves a transaction to the DB
func (app *application) SaveTransaction(txn models.Transaction) (int, error) {
	id, err := app.DB.InsertTransaction(txn)
	if err != nil {
		app.errorLog.Println(err)
		return 0, err
	}

	return id, nil
}

//SaveOrder saves an order to the DB
func (app *application) SaveOrder(order models.Order) (int, error) {
	id, err := app.DB.InsertOrder(order)
	if err != nil {
		app.errorLog.Println(err)
		return 0, err
	}

	return id, nil
}

func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {

	var userInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	//get the user from the db by email, send error if invalid email
	user, err := app.DB.GetUserByEmail(userInput.Email)
	if err != nil {
		log.Println(err)
		app.invalidCredentials(w)
		return
	}

	//validate the password
	validPassword, err := app.matchPasswords(user.Password, userInput.Password)
	if err != nil {
		log.Println(err)
		app.invalidCredentials(w)
		return
	}

	if !validPassword {
		log.Println(err)
		app.invalidCredentials(w)
		return
	}
	//generate the token
	token, err := models.GenerateToken(user.ID, 24*time.Hour, models.ScopeAuthentication)
	if err != nil {
		log.Println(err)
		app.badRequest(w, r, err)
		return
	}

	//save to DB
	err = app.DB.InsertToken(token, user)
	if err != nil {
		log.Println(err)
		app.badRequest(w, r, err)
		return
	}

	//send a response
	var payload struct {
		Error   bool          `json:"error"`
		Message string        `json:"message"`
		Token   *models.Token `json:"authToken"`
	}

	payload.Error = false
	payload.Message = fmt.Sprintf("token for %s created ", userInput.Email)
	payload.Token = token

	err = app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

}

func (app *application) authenticateToken(r *http.Request) (*models.User, error) {

	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header")
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no authorization header")
	}

	token := headerParts[1]
	if len(token) != 26 {
		return nil, errors.New("auth token wrong size")
	}

	log.Println(token)
	//get the user from tokens table
	user, err := app.DB.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no matching user found")
	}
	return user, nil
}
func (app *application) CheckAuthenticated(w http.ResponseWriter, r *http.Request) {
	//validate token and get user
	user, err := app.authenticateToken(r)
	if err != nil {
		app.invalidCredentials(w)
	}
	//valid user
	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	payload.Error = false
	payload.Message = fmt.Sprintf("authenticated user %s", user.Email)

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) Register(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Password    string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)
	if err != nil {
		log.Println(err)
		app.badRequest(w, r, err)
		return
	}

	var newUser models.User
	newUser.FirstName = userInput.FirstName
	newUser.LastName = userInput.LastName
	newUser.Email = userInput.Email
	newUser.DisplayName = userInput.DisplayName
	newUser.Password = userInput.Password

	id, err := app.DB.InsertUser(newUser)
	if err != nil {
		log.Println(err)
		app.badRequest(w, r, err)
		return
	}

	newUser.ID = id
	var payload struct {
		Error   bool        `json:"error"`
		Message string      `json:"message"`
		User    models.User `json:"newUser"`
	}

	payload.Error = false
	payload.Message = fmt.Sprintf("New user %s", userInput.Email)
	payload.User = newUser
	err = app.writeJSON(w, http.StatusCreated, payload)
	if err != nil {
		log.Println(err)
		app.badRequest(w, r, err)
		return
	}
}

func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	var txnData struct {
		PaymentAmount   int    `json:"amount"`
		PaymentCurrency string `json:"currency"`
		FirstName       string `json:"firstName"`
		LastName        string `json:"lastName"`
		Email           string `json:"email"`
		PaymentIntent   string `json:"paymentIntent"`
		PaymentMethod   string `json:"paymentMethod"`
		ExpiryMonth     int    `json:"expiryMonth"`
		ExpiryYear      int    `json:"expiryYear"`
		BankReturnCode  string `json:"bankReturnCode"`
		LastFour        string `json:"lastFour"`
	}

	err := app.readJSON(w, r, &txnData)
	if err != nil {
		log.Print(err)
		app.badRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret: app.config.stripe.secretKey,
		Key:    app.config.stripe.key,
	}

	pi, err := card.RetrievePaymentIntent(txnData.PaymentIntent)
	if err != nil {
		log.Print(err)
		app.badRequest(w, r, err)
		return
	}

	pm, err := card.GetPaymentMethod(txnData.PaymentMethod)
	if err != nil {
		log.Print(err)
		app.badRequest(w, r, err)
		return
	}

	txnData.LastFour = pm.Card.Last4
	txnData.ExpiryMonth = int(pm.Card.ExpMonth)
	txnData.ExpiryMonth = int(pm.Card.ExpYear)

	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		BankReturnCode:      pi.Charges.Data[0].ID,
		PaymentIntent:       txnData.PaymentIntent,
		PaymentMethod:       txnData.PaymentMethod,
		TransactionStatusID: 2,
	}

	_, err = app.SaveTransaction(txn)
	if err != nil {
		log.Print(err)
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, txn)
}

func (app *application) SendPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		log.Print(err)
		app.badRequest(w, r, err)
		return
	}

	//verify that email exists
	_, err = app.DB.GetUserByEmail(payload.Email)

	if err != nil {
		var resp struct {
			Error   bool   `json:"error"`
			Message string `json:"message"`
		}

		resp.Error = true
		resp.Message = "No matching email found"

		app.writeJSON(w, http.StatusAccepted, resp)
		return
	}

	link := fmt.Sprintf("%s/reset-password?email=%s", app.config.frontend, payload.Email)

	sign := urlsigner.Signer{
		Secret: []byte(app.config.secretKey),
	}

	signedLink := sign.GenerateTokenFromString(link)

	var data struct {
		Link string
	}

	data.Link = signedLink

	//send mail
	err = app.SendMail("info@widgets.com", payload.Email, "Password Reset Request", "password-reset", data)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "Reset email successfully sent."

	app.writeJSON(w, http.StatusCreated, resp)
}

func (app *application) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	encryptor := encryption.Encryption{
		Key: []byte(app.config.secretKey),
	}

	realEmail, err := encryptor.Decrypt(payload.Email)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	user, err := app.DB.GetUserByEmail(realEmail)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	err = app.DB.UpdatePasswordForUser(user, string(newHash))
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "Password changed"

	app.writeJSON(w, http.StatusCreated, resp)
}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	allSales, err := app.DB.GetAllOrders()
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, allSales)
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	allSales, err := app.DB.GetAllSubscriptions()
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, allSales)
}

func (app *application) GetSale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(id)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	order, err := app.DB.GetOrderByID(orderID)
	if err != nil {
		app.errorLog.Print(err)
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, order)
}
