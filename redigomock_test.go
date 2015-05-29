package redigomock

import (
	"fmt"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Person struct {
	Name string `redis:"name"`
	Age  int    `redis:"age"`
}

func RetrievePerson(conn redis.Conn, id string) (Person, error) {
	var person Person

	values, err := redis.Values(conn.Do("HGETALL", fmt.Sprintf("person:%s", id)))
	if err != nil {
		return person, err
	}

	err = redis.ScanStruct(values, &person)
	return person, err
}

func RetrievePeople(conn redis.Conn, ids []string) ([]Person, error) {
	var people []Person

	for _, id := range ids {
		conn.Send("HGETALL", fmt.Sprintf("person:%s", id))
	}

	for i := 0; i < len(ids); i++ {
		values, err := redis.Values(conn.Receive())
		if err != nil {
			return nil, err
		}

		var person Person
		err = redis.ScanStruct(values, &person)
		if err != nil {
			return nil, err
		}

		people = append(people, person)
	}

	return people, nil
}

func TestDoCommand(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})

	person, err := RetrievePerson(connection, "1")
	if err != nil {
		t.Fatal(err)
	}

	if person.Name != "Mr. Johson" {
		t.Errorf("Invalid name. Expected 'Mr. Johson' and got '%s'", person.Name)
	}

	if person.Age != 42 {
		t.Errorf("Invalid age. Expected '42' and got '%d'", person.Age)
	}
}

func TestDoCommandMultipleReturnValues(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	}).ExpectMap(map[string]string{
		"name": "Ms. Jennifer",
		"age":  "28",
	}).ExpectError(fmt.Errorf("simulated error"))

	person, err := RetrievePerson(connection, "1")
	if err != nil {
		t.Fatal(err)
	}
	if person.Name != "Mr. Johson" {
		t.Errorf("Invalid name. Expected 'Mr. Johson' and got '%s'", person.Name)
	}
	if person.Age != 42 {
		t.Errorf("Invalid age. Expected '42' and got '%d'", person.Age)
	}

	person, err = RetrievePerson(connection, "1")
	if err != nil {
		t.Fatal(err)
	}
	if person.Name != "Ms. Jennifer" {
		t.Errorf("Invalid name. Expected 'Mr. Johson' and got '%s'", person.Name)
	}
	if person.Age != 28 {
		t.Errorf("Invalid age. Expected '28' and got '%d'", person.Age)
	}

	_, err = RetrievePerson(connection, "1")
	if err == nil {
		t.Error("Should return an error!")
	}
}

func TestDoGenericCommand(t *testing.T) {
	connection := NewConn()

	connection.GenericCommand("HGETALL").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})

	person, err := RetrievePerson(connection, "1")
	if err != nil {
		t.Fatal(err)
	}

	if person.Name != "Mr. Johson" {
		t.Errorf("Invalid name. Expected 'Mr. Johson' and got '%s'", person.Name)
	}

	if person.Age != 42 {
		t.Errorf("Invalid age. Expected '42' and got '%d'", person.Age)
	}
}

func TestDoCommandWithGeneric(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})

	connection.GenericCommand("HGETALL").ExpectMap(map[string]string{
		"name": "Mr. Mark",
		"age":  "32",
	})

	person, err := RetrievePerson(connection, "1")
	if err != nil {
		t.Fatal(err)
	}

	if person.Name != "Mr. Johson" {
		t.Errorf("Invalid name. Expected 'Mr. Johson' and got '%s'", person.Name)
	}

	if person.Age != 42 {
		t.Errorf("Invalid age. Expected '42' and got '%d'", person.Age)
	}
}

func TestDoCommandWithError(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectError(fmt.Errorf("simulated error"))

	_, err := RetrievePerson(connection, "1")
	if err == nil {
		t.Error("Should return an error!")
		return
	}
}

func TestDoCommandWithUnexpectedCommand(t *testing.T) {
	connection := NewConn()

	_, err := RetrievePerson(connection, "X")
	if err == nil {
		t.Error("Should detect a command not registered!")
		return
	}
}

func TestDoCommandWithoutResponse(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1")

	_, err := RetrievePerson(connection, "1")
	if err == nil {
		t.Fatal("Returning an information when it shoudn't")
	}
}

func TestSendFlushReceive(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})
	connection.Command("HGETALL", "person:2").ExpectMap(map[string]string{
		"name": "Ms. Jennifer",
		"age":  "28",
	})

	people, err := RetrievePeople(connection, []string{"1", "2"})
	if err != nil {
		t.Fatal(err)
	}

	if len(people) != 2 {
		t.Errorf("Wrong number of people. Expected '2' and got '%d'", len(people))
	}

	if people[0].Name != "Mr. Johson" || people[1].Name != "Ms. Jennifer" {
		t.Error("People name order are wrong")
	}

	if people[0].Age != 42 || people[1].Age != 28 {
		t.Error("People age order are wrong")
	}

	if _, err := connection.Receive(); err == nil {
		t.Error("Not detecting when there's no more items to receive")
	}
}

func TestSendReceiveWithWait(t *testing.T) {
	conn := Conn{
		ReceiveWait: true,
		ReceiveNow:  make(chan bool),
	}

	conn.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})
	conn.Command("HGETALL", "person:2").ExpectMap(map[string]string{
		"name": "Ms. Jennifer",
		"age":  "28",
	})

	ids := []string{"1", "2"}
	for _, id := range ids {
		conn.Send("HGETALL", fmt.Sprintf("person:%s", id))
	}

	var people []Person

	go func() {
		for i := 0; i < len(ids); i++ {
			values, err := redis.Values(conn.Receive())
			if err != nil {
				t.Fatal(err)
			}

			var person Person
			err = redis.ScanStruct(values, &person)
			if err != nil {
				t.Fatal(err)
			}

			people = append(people, person)
		}
	}()

	for i := 0; i < len(ids); i++ {
		conn.ReceiveNow <- true
	}
	time.Sleep(10 * time.Millisecond)

	if len(people) != 2 {
		t.Fatalf("Wrong number of people. Expected '2' and got '%d'", len(people))
	}

	if people[0].Name != "Mr. Johson" || people[1].Name != "Ms. Jennifer" {
		t.Error("People name order are wrong")
	}

	if people[0].Age != 42 || people[1].Age != 28 {
		t.Error("People age order are wrong")
	}
}

func TestSendFlushReceiveWithError(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})
	connection.Command("HGETALL", "person:2").ExpectMap(map[string]string{
		"name": "Ms. Jennifer",
		"age":  "28",
	})
	connection.Command("HGETALL", "person:2").ExpectError(fmt.Errorf("simulated error"))

	_, err := RetrievePeople(connection, []string{"1", "2", "3"})
	if err == nil {
		t.Error("Not detecting error when using send/flush/receive")
	}
}

func TestDummyFunctions(t *testing.T) {
	var conn Conn

	if conn.Close() != nil {
		t.Error("Close is not dummy!")
	}

	conn.CloseMock = func() error {
		return fmt.Errorf("close error")
	}

	if err := conn.Close(); err == nil || err.Error() != "close error" {
		t.Errorf("Not mocking Close method correctly. Expected “close error” and got “%v”", err)
	}

	if conn.Err() != nil {
		t.Error("Err is not dummy!")
	}

	conn.ErrMock = func() error {
		return fmt.Errorf("err error")
	}

	if err := conn.Err(); err == nil || err.Error() != "err error" {
		t.Errorf("Not mocking Err method correctly. Expected “err error” and got “%v”", err)
	}

	if conn.Flush() != nil {
		t.Error("Flush is not dummy!")
	}

	conn.FlushMock = func() error {
		return fmt.Errorf("flush error")
	}

	if err := conn.Flush(); err == nil || err.Error() != "flush error" {
		t.Errorf("Not mocking Flush method correctly. Expected “flush error” and got “%v”", err)
	}
}

func TestClear(t *testing.T) {
	connection := NewConn()

	connection.Command("HGETALL", "person:1").ExpectMap(map[string]string{
		"name": "Mr. Johson",
		"age":  "42",
	})
	connection.Command("HGETALL", "person:2").ExpectMap(map[string]string{
		"name": "Ms. Jennifer",
		"age":  "28",
	})
	connection.GenericCommand("HGETALL").ExpectMap(map[string]string{
		"name": "Ms. Mark",
		"age":  "32",
	})

	connection.Send("HGETALL", "person:1")
	connection.Send("HGETALL", "person:2")

	connection.Clear()

	if len(connection.commands) > 0 {
		t.Error("Clear function not clearing registered commands")
	}

	if len(connection.queue) > 0 {
		t.Error("Clear function not clearing the queue")
	}
}
