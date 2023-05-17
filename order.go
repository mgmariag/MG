package orders

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/rogpeppe/go-internal/lockedfile"
)

const (
	ORDER_READY     = "ready"
	ORDER_TAKEN     = "taken"
	ORDER_PREPARING = "preparing"
)

var (
	ErrOrderNotExists = errors.New("order does not exist")
	ErrOrderNotReady  = errors.New("order is not ready")
)

type Order struct {
	Id      uint32 `json:"id"`
	Status  string `json:"status"`
	Content string `json:"content"`
}

type Orders = map[uint32]Order

// OrderDB represents the orders database. Fields are exported to allow gob
// encoding without manually implementing GobEncoder and GobDecored. DO NOT
// modify the fields directly!
type OrderDB struct {
	// id is the key, which is redundant but used for fast reads. I know I
	// could've dropped the id from order but who the fuck cares
	file    string
	Updates chan string
}

/*
	Note that panics below are used for debugging purposes.
*/

// open opens the db for reading.
func (o *OrderDB) open() (*lockedfile.File, Orders) {
	f, err := lockedfile.Open(o.file)
	if err != nil {
		panic(err)
	}
	var orders = make(Orders)
	data, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(data, &orders)
	return f, orders
}

// openfile opens the db for writing.
func (o *OrderDB) openfile() (*lockedfile.File, Orders) {
	f, err := lockedfile.OpenFile(o.file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	var orders = make(Orders)
	data, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(data, &orders)
	return f, orders
}

// flush writes the database to disk.
func (o *OrderDB) flush(w *lockedfile.File, orders Orders) {
	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		panic(err)
	}

	// Ensures that the contents of the file are erased, if opened in RDWR mode.
	w.Truncate(0)
	_, err = w.WriteAt(data, 0)
	if err != nil {
		panic(err)
	}
	if o.Updates != nil {
		o.Updates <- string(data)
	}
}

func (o *OrderDB) String() string {
	f, orders := o.open()
	defer f.Close()
	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (o *OrderDB) AddOrder(order Order) {
	f, orders := o.openfile()
	defer f.Close()
	orders[order.Id] = order
	o.flush(f, orders)
}

func (o *OrderDB) TakeOrder(oid uint32) (Order, error) {
	f, orders := o.openfile()
	defer f.Close()
	retrieved, ok := orders[oid]
	if !ok {
		return Order{}, ErrOrderNotExists
	}
	if retrieved.Status != ORDER_READY {
		return Order{}, ErrOrderNotReady
	}
	delete(orders, oid)
	o.flush(f, orders)
	return retrieved, nil
}

func (o *OrderDB) ChangeOrderStatus(oid uint32, status string) {
	// Technically, this doesn't require a lock, but remains to be seen.
	f, orders := o.openfile()
	defer f.Close()
	order := orders[oid]
	order.Status = status
	orders[oid] = order
	o.flush(f, orders)
}

func NewOrdersDB(file string, c chan string) OrderDB {
	return OrderDB{
		file:    file,
		Updates: c,
	}
}

// //+
func (o *OrderDB) GetOrders() ([]Order, error) {
	data, err := ioutil.ReadFile(o.file)
	if err != nil {
		return nil, err
	}

	var orders map[string]Order
	err = json.Unmarshal(data, &orders)
	if err != nil {
		return nil, err
	}

	var orderList []Order
	for _, order := range orders {
		orderList = append(orderList, order)
	}

	return orderList, nil
}
