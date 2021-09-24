package models

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//DBModel is the type for DB connection
type DBModel struct {
	DB *sql.DB
}

//Models is the wrapper for all models
type Models struct {
	DB DBModel
}

//NewModels returns a model with DB connection pool
func NewModels(db *sql.DB) Models {
	return Models{
		DB: DBModel{DB: db},
	}
}

//Widget type for all widgets
type Widget struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventoryLevel"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	IsRecurring    bool      `json:"isRecurring"`
	PlanID         string    `json:"planId"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

//Order is the type for all orders
type Order struct {
	ID            int         `json:"id"`
	WidgetID      int         `json:"widgetId"`
	TransactionID int         `json:"transactionId"`
	CustomerID    int         `json:"customerId"`
	StatusID      int         `json:"statusId"`
	Quantity      int         `json:"quantity"`
	Amount        int         `json:"amount"`
	Widget        Widget      `json:"widget"`
	Transaction   Transaction `json:"transaction"`
	Customer      Customer    `json:"customer"`
	CreatedAt     time.Time   `json:"-"`
	UpdatedAt     time.Time   `json:"-"`
}

//Status is for statuses
type Status struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

//TransactionStatus is for transaction statuses
type TransactionStatus struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

//Transaction is for transactions
type Transaction struct {
	ID                  int       `json:"id"`
	Currency            string    `json:"currency"`
	Amount              int       `json:"amount"`
	LastFour            string    `json:"lastFour"`
	ExpiryMonth         int       `json:"expiry_month"`
	ExpiryYear          int       `json:"expiry_year"`
	BankReturnCode      string    `json:"bankReturnCode"`
	PaymentIntent       string    `json:"paymentIntent"`
	PaymentMethod       string    `json:"paymentMethod"`
	TransactionStatusID int       `json:"transactionStatusId"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}

//User is for users
type User struct {
	ID          int       `json:"id"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Password    string    `json:"-"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

//Customer is for customers
type Customer struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (m *DBModel) GetWidget(id int) (Widget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var widget Widget

	row := m.DB.QueryRowContext(ctx, `
	select 
		id, name, description, inventory_level, price, coalesce(image, ""),
		is_recurring, plan_id,
		created_at, updated_at
	from 
		widgets 
	where id = ?`, id)
	err := row.Scan(
		&widget.ID,
		&widget.Name,
		&widget.Description,
		&widget.InventoryLevel,
		&widget.Price,
		&widget.Image,
		&widget.IsRecurring,
		&widget.PlanID,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	)

	if err != nil {
		return widget, err
	}

	return widget, nil
}

//InsertTransaction inserts a new transaction and return its id
func (m *DBModel) InsertTransaction(txn Transaction) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		insert into transactions
		(amount, currency, last_four, bank_return_code, expiry_month, expiry_year,
			payment_intent, payment_method,
		transaction_status_id, created_at, updated_at)
		values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := m.DB.ExecContext(ctx, stmt,
		txn.Amount,
		txn.Currency,
		txn.LastFour,
		txn.BankReturnCode,
		txn.ExpiryMonth,
		txn.ExpiryYear,
		txn.PaymentIntent,
		txn.PaymentMethod,
		txn.TransactionStatusID,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		log.Print(err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Print(err)
		return 0, err
	}

	return int(id), nil
}

//InsertOrder inserts a new order and return its id
func (m *DBModel) InsertOrder(order Order) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		insert into orders
		(widget_id, transaction_id, status_id, quantity, customer_id,
		amount, created_at, updated_at)
		values(?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := m.DB.ExecContext(ctx, stmt,
		order.WidgetID,
		order.TransactionID,
		order.StatusID,
		order.Quantity,
		order.CustomerID,
		order.Amount,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		log.Print(err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Print(err)
		return 0, err
	}

	return int(id), nil
}

//InsertCustomer inserts a new customer and return its id
func (m *DBModel) InsertCustomer(c Customer) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		insert into customers
		(first_name, last_name, email, created_at, updated_at)
		values(?, ?, ?, ?, ?)
	`

	result, err := m.DB.ExecContext(ctx, stmt,
		c.FirstName,
		c.LastName,
		c.Email,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		log.Print(err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Print(err)
		return 0, err
	}

	return int(id), nil
}

//InsertCustomer inserts a new customer and return its id
func (m *DBModel) InsertUser(u User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	password := []byte(u.Password)

	// Create a bcrypt hash of the plain-text password.
	hashedPassword, err := bcrypt.GenerateFromPassword(password, 12)
	if err != nil {
		log.Print(err)
		return 0, err
	}

	stmt := `
		insert into users
		(first_name, last_name, email, display_name, password, created_at, updated_at)
		values(?, ?, ?, ?, ?, ?, ?)
	`

	result, err := m.DB.ExecContext(ctx, stmt,
		u.FirstName,
		u.LastName,
		u.Email,
		u.DisplayName,
		hashedPassword,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		log.Print(err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Print(err)
		return 0, err
	}

	return int(id), nil
}

//GetUserByEmail gets a user by email address
func (m *DBModel) GetUserByEmail(email string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email = strings.ToLower(email)
	var u User

	row := m.DB.QueryRowContext(ctx, `
		select 
			id, first_name, last_name, email, password, display_name, created_at, updated_at
		from
			users
		where email = ?
	`, email)

	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.Password,
		&u.DisplayName,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		return u, err
	}

	return u, nil

}

func (m *DBModel) Authenticate(email, password string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	row := m.DB.QueryRowContext(ctx, "select id, password from users where email = ?", email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return id, err
	}

	//found a user
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, errors.New("incorrect password")
	} else if err != nil {
		return 0, err
	}

	return id, nil

}

//UpdatePasswordForUser resets a user's password
func (m *DBModel) UpdatePasswordForUser(u User, newPW string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `update users set password = ? where id = ?`

	_, err := m.DB.ExecContext(ctx, stmt, newPW, u.ID)
	if err != nil {
		return err
	}

	return nil

}

func (m *DBModel) GetAllOrders() ([]*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, o.status_id,
		o.quantity, o.amount, o.created_at, o.updated_at, w.id, w.name,
		t.id, t.amount, t.currency, t.last_four, t.expiry_month, t.expiry_year,
		t.payment_intent, t.bank_return_code, c.id, c.first_name, c.last_name, c.email
	from 
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 0
	order by
		o.created_at desc
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)

		if err != nil {
			return nil, err
		}

		orders = append(orders, &o)
	}

	return orders, nil
}

func (m *DBModel) GetAllSubscriptions() ([]*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, o.status_id,
		o.quantity, o.amount, o.created_at, o.updated_at, w.id, w.name,
		t.id, t.amount, t.currency, t.last_four, t.expiry_month, t.expiry_year,
		t.payment_intent, t.bank_return_code, c.id, c.first_name, c.last_name, c.email
	from 
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 1
	order by
		o.created_at desc
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)

		if err != nil {
			return nil, err
		}

		orders = append(orders, &o)
	}

	return orders, nil
}

func (m *DBModel) GetOrderByID(id int) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var o Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, o.status_id,
		o.quantity, o.amount, o.created_at, o.updated_at, w.id, w.name,
		t.id, t.amount, t.currency, t.last_four, t.expiry_month, t.expiry_year,
		t.payment_intent, t.bank_return_code, c.id, c.first_name, c.last_name, c.email
	from 
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		o.id = ?
	`

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&o.ID,
		&o.WidgetID,
		&o.TransactionID,
		&o.CustomerID,
		&o.StatusID,
		&o.Quantity,
		&o.Amount,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Widget.ID,
		&o.Widget.Name,
		&o.Transaction.ID,
		&o.Transaction.Amount,
		&o.Transaction.Currency,
		&o.Transaction.LastFour,
		&o.Transaction.ExpiryMonth,
		&o.Transaction.ExpiryYear,
		&o.Transaction.PaymentIntent,
		&o.Transaction.BankReturnCode,
		&o.Customer.ID,
		&o.Customer.FirstName,
		&o.Customer.LastName,
		&o.Customer.Email,
	)

	if err != nil {
		return o, err
	}

	return o, nil
}
