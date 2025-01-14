package library

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Note(sn): create valid and invalid examples here and share between tests.

func assertContentType(t testing.TB, response *httptest.ResponseRecorder,
	want, warningMessage string) {
	t.Helper()
	if response.Result().Header.Get("content-type") != want {
		t.Errorf("response did not have content-type of %s, got %v", want,
			response.Result().Header)
	}
}

func assertNoError(t testing.TB, got, want string) {
	t.Helper()

	if got != "" {
		t.Errorf("got error %q did not want any error message", got)
	}
}

func assertError(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got error %q want %q", got, want)
	}
}

func assertStatus(t testing.TB, got, want int, warningMessage string) {
	t.Helper()
	if got != want {
		t.Errorf("did not get correct status, got %d, want %d", got, want)
	}
}

func assertDeletedBook(t *testing.T, isbn string, db *sql.DB, usage string) {
	t.Helper()
	book := FindSpecificBook(db, isbn)
	if (book != Book{}) {
		t.Errorf("The book with the isbn %q should have been deleted", isbn)
	}
}

func assertEqualBook(t *testing.T, got, wanted Book, warningMessage string) {
	t.Helper()
	if got.ISBN != wanted.ISBN || got.Author.FirstName != wanted.Author.FirstName ||
		got.Title != wanted.Title || got.Author.LastName != wanted.Author.LastName ||
		got.Publisher != wanted.Publisher {
		t.Errorf("got %v want %v", got, wanted)
	}
}

func assertEqualBooks(t *testing.T, got, wanted []Book, warningMessage string) {
	t.Helper()
	for i, _ := range got {
		if got[i].ISBN != wanted[i].ISBN || got[i].Author.FirstName !=
			wanted[i].Author.FirstName || got[i].Title != wanted[i].Title ||
			got[i].Author.LastName != wanted[i].Author.LastName ||
			got[i].Publisher != wanted[i].Publisher {
			t.Errorf("got %v want %v", got, wanted)
		}
	}
}

func createTempDatabase(t *testing.T) (*sql.DB, func() error) {
	t.Helper()
	tempFile, err := os.CreateTemp("", "")
	require.NoError(t, err)
	db, err := sql.Open("sqlite", tempFile.Name())
	require.NoError(t, err)
	require.NoError(t, EnsureSchema(db))
	cleanup := func() error {
		return os.Remove(tempFile.Name()) // Removes the temporary file
	}
	return db, cleanup
}

func createNewRequest(
	httpMethod, urlPath string,
	jsonBytes []byte,
	db *sql.DB,
) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(httpMethod, urlPath,
		bytes.NewReader(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	NewServer(db).ServeHTTP(response, request)
	return response
}

func TestCREATEBookMETHOD(t *testing.T) {
	db, cleanup := createTempDatabase(t)
	defer cleanup()

	t.Run("Creates a book and stores it in the library", func(t *testing.T) {
		///Arange
		isbn := "1233211233215"
		want := Book{
			ISBN:  isbn,
			Title: "star wars",
			Author: &Author{
				FirstName: "george",
				LastName:  "lucas"},
			Publisher: "adlibris"}
		dataInfo := &want

		// Note(sn): require.NoError(t, err)
		jsonBytes, _ := json.Marshal(dataInfo)

		// Act
		response := createNewRequest(http.MethodPost,
			"/api/books/"+isbn, jsonBytes, db)
		got := FindSpecificBook(db, isbn)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json"+
			"content type application/json")
		assertStatus(t, response.Code, http.StatusOK, "Should get status code 200:"+
			"status OK")
		assertEqualBook(t, got, want, "Should be equal")
	})

	t.Run("Creates a book that already exists in the library", func(t *testing.T) {
		// Arange
		isbn := "1233211233215"
		want := Book{
			ISBN:  isbn,
			Title: "star wars the revenge of the sith",
			Author: &Author{
				FirstName: "george",
				LastName:  "lucas"},
			Publisher: "adlibris new publisher"}
		dataInfo := &want
		jsonBytes, _ := json.Marshal(dataInfo)

		// Act
		response := createNewRequest(http.MethodPost,
			"/api/books/"+isbn, jsonBytes, db)
		b, _ := ioutil.ReadAll(response.Body)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json"+
			" content type application/json")
		assertStatus(t, response.Code, http.StatusConflict, "Should get status"+
			" code 409: status conflict")
		assertError(t, string(b), "A book with this ISBN already exits")
	})

	t.Run("Creates a new book and sets the time parameter", func(t *testing.T) {
		// Arange
		isbn := "1233211233218"
		want := Book{
			ISBN:       isbn,
			Title:      "star wars the revenge of the sith",
			CreateTime: time.Now(),
			Author: &Author{
				FirstName: "george",
				LastName:  "lucas"},
			Publisher: "adlibris new publisher"}
		dataInfo := &want
		jsonBytes, _ := json.Marshal(dataInfo)

		// Act
		response := createNewRequest(http.MethodPost,
			"/api/books/"+isbn, jsonBytes, db)
		b, _ := ioutil.ReadAll(response.Body)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json"+
			" content type application/json")
		assertStatus(t, response.Code, http.StatusForbidden, "Should get status"+
			" code 403: status forbidden")
		assertError(t, string(b), "Not allowed to change CreateTime or UpdateTime")
	})

	t.Run("Creates a new book with isbn on the wrong format", func(t *testing.T) {
		// Arange
		isbn := "123321123321a"
		want := Book{
			ISBN:  isbn,
			Title: "star wars the revenge of the sith",
			Author: &Author{
				FirstName: "george",
				LastName:  "lucas"},
			Publisher: "adlibris new publisher"}
		dataInfo := &want

		jsonBytes, _ := json.Marshal(dataInfo)

		// Act
		response := createNewRequest(http.MethodPost,
			"/api/books/"+isbn, jsonBytes, db)
		b, _ := ioutil.ReadAll(response.Body)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json"+
			" content type application/json")
		assertStatus(t, response.Code, http.StatusNotAcceptable, "Should get status"+
			" code 406: status forbidden")
		assertError(t, string(b), "validation failed, field error(s):"+
			" isbn . Fix these error before proceeding")
	})
}

func TestGETBooksMETHOD(t *testing.T) { //List
	db, cleanup := createTempDatabase(t)
	defer cleanup()

	t.Run("Creates two book instances and stores it in the library database",
		func(t *testing.T) {
			/// A new book
			isbn := "1233211233215"
			want := Book{
				ISBN:  isbn,
				Title: "star wars",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas",
				},
				Publisher: "adlibris",
			}
			dataInfo := &want

			jsonBytes, err := json.Marshal(dataInfo)
			if err != nil {
				t.Fatal(err)
			}

			// Act
			_ = createNewRequest(http.MethodPost,
				"/api/books/"+isbn, jsonBytes, db)

			//New book
			isbn2 := "1233211233213"
			want2 := Book{
				ISBN:  isbn2,
				Title: "star wars revenge of the sith",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas"},
				Publisher: "adlibris"}
			dataInfo2 := &want2

			jsonBytes2, err2 := json.Marshal(dataInfo2)
			if err2 != nil {
				t.Fatal(err)
			}

			// Act
			_ = createNewRequest(http.MethodPost,
				"/api/books/"+isbn2, jsonBytes2, db)

		})

	t.Run("gets all the books in the library database", func(t *testing.T) {
		// Arange
		response := createNewRequest(http.MethodGet,
			"/api/books", nil, db)
		want := ReadDatabaseList(db)

		//act
		var got []Book
		_ = json.NewDecoder(response.Body).Decode(&got) // Act

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json "+
			"content type application/json")
		assertStatus(t, response.Code, http.StatusOK, "Should get status "+
			"code 200: status OK")
		assertEqualBooks(t, got, want, "Should be equal")
	})
	/*
		t.Run("get a specific book in the library", func(t *testing.T) {
			// Arange
			isbn := "1233211233213"
			request, _ := http.NewRequest(http.MethodGet, "/api/books/"+isbn, nil)
			response := httptest.NewRecorder()
			NewServer(db).ServeHTTP(response, request)
			want := FindSpecificBook(db, isbn)

			var got Book
			err := json.NewDecoder(response.Body).Decode(&got) // Act

			//assert
			assertContentType(t, response, jsonContentType, "Should have the json content type application/json")
			assertNoError(t, err.Error(), "Should have no errors")
			assertStatus(t, response.Code, http.StatusOK, "Should get status code 200: status OK")
			assertEqualBook(t, got, want, "Should be equal")
		})

		t.Run("get a book that does not exist in the library", func(t *testing.T) {
			// Arange
			isbn := "1233211233216"
			request, _ := http.NewRequest(http.MethodGet, "/api/books/"+isbn, nil)
			response := httptest.NewRecorder()
			NewServer(db).ServeHTTP(response, request)

			var got Book
			err := json.NewDecoder(response.Body).Decode(&got) // Act

			//assert
			assertContentType(t, response, jsonContentType, "Should have the json content type application/json")
			assertError(t, err.Error(), "invalid character 'T' looking for beginning of value")
			assertStatus(t, response.Code, http.StatusNotFound, "Should have status code 404: statusNotFound")
		})*/
}

func TestDELETEBookMETHOD(t *testing.T) { //List
	t.Parallel()
	db, cleanup := createTempDatabase(t)
	defer cleanup()

	t.Run("Creates two book instances and stores it in the library database",
		func(t *testing.T) {
			/// A new book
			isbn := "1233211233215"
			want := Book{
				ISBN:  isbn,
				Title: "star wars",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas"},
				Publisher: "adlibris"}
			dataInfo := &want

			jsonBytes, _ := json.Marshal(dataInfo)

			// Act
			_ = createNewRequest(http.MethodPost,
				"/api/books/"+isbn, jsonBytes, db)

			//New book
			isbn2 := "1233211233213"
			want2 := Book{
				ISBN:  isbn2,
				Title: "star wars revenge of the sith",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas"},
				Publisher: "adlibris"}
			dataInfo2 := &want2

			jsonBytes2, _ := json.Marshal(dataInfo2)

			// Act
			_ = createNewRequest(http.MethodPost,
				"/api/books/"+isbn2, jsonBytes2, db)

		})

	t.Run("Delete a book that does exist in the library", func(t *testing.T) {
		// Arange
		isbn := "1233211233213"
		response := createNewRequest(http.MethodDelete,
			"/api/books/"+isbn, nil, db)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json "+
			"content type application/json")
		assertStatus(t, response.Code, http.StatusOK, "Should have status"+
			"code 200: status OK")
		assertDeletedBook(t, isbn, db, "Checks if the book is deleted from "+
			"the database")
	})

	t.Run("Delete a book that does not exist in the library", func(t *testing.T) {
		// Arange
		isbn := "1233211233210"
		response := createNewRequest(http.MethodDelete,
			"/api/books/"+isbn, nil, db)
		b, _ := ioutil.ReadAll(response.Body)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json "+
			"content type application/json")
		assertStatus(t, response.Code, http.StatusNotFound, "Should have status "+
			"code 404: statusNotFound")
		assertDeletedBook(t, isbn, db, "Checks if the book is deleted from "+
			"the database")
		assertError(t, string(b), "The book did not exist in the library or "+
			"was already deleted")
	})

}

func TestUpdateBooks(t *testing.T) {
	db, cleanup := createTempDatabase(t)
	defer cleanup()

	t.Run("Creates a book instances and stores it in the library database",
		func(t *testing.T) {
			/// A new book
			isbn := "1233211233215"
			want := Book{
				ISBN:  isbn,
				Title: "star wars",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas"},
				Publisher: "adlibris"}
			dataInfo := &want
			jsonBytes, _ := json.Marshal(dataInfo)

			// Act
			_ = createNewRequest(http.MethodPost,
				"/api/books/"+isbn, jsonBytes, db)

		})

	t.Run("Updates a specific book which exists in the library",
		func(t *testing.T) {
			// Arange
			isbn := "1233211233215"
			want := Book{
				ISBN:  isbn,
				Title: "star wars phantom menance",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas"},
				Publisher: "adlibris"}
			dataInfo := &want
			jsonBook, _ := json.Marshal(dataInfo)

			//act
			response := createNewRequest(http.MethodPut,
				"/api/books/"+isbn, jsonBook, db)

			var got Book
			_ = json.NewDecoder(response.Body).Decode(&got) // Act

			//assert
			assertContentType(t, response, jsonContentType, "Should have the json "+
				"content type application/json")
			assertStatus(t, response.Code, http.StatusOK, "Should jave status "+
				"code 200: status OK")
			assertEqualBook(t, got, want, "Should be equal")

			//TODO Något med error blir fel här?

		})

	t.Run("Updates a specific book that does not exists in the library",
		func(t *testing.T) {
			// Arange
			isbn := "1233211233210"
			want := Book{
				ISBN:  isbn,
				Title: "star wars phantom menance",
				Author: &Author{
					FirstName: "george",
					LastName:  "lucas"},
				Publisher: "adlibris"}
			dataInfo := &want
			jsonBook, _ := json.Marshal(dataInfo)

			//act
			response := createNewRequest(http.MethodPut,
				"/api/books/"+isbn, jsonBook, db)
			b, _ := ioutil.ReadAll(response.Body)

			//assert
			assertContentType(t, response, jsonContentType, "Should have the json "+
				"content type application/json")
			assertStatus(t, response.Code, http.StatusNotFound, "Should jave status "+
				"code 200: status OK")
			assertError(t, string(b), "The book did not exist in the library")
		})

	t.Run("changing the ISBN which is not allowed ", func(t *testing.T) {
		// Arange
		isbn := "1233211233215"
		want := Book{
			ISBN:  "1233211233210",
			Title: "star wars phantom menance",
			Author: &Author{
				FirstName: "george",
				LastName:  "lucas"},
			Publisher: "adlibris"}
		dataInfo := &want
		jsonBook, _ := json.Marshal(dataInfo)

		//act
		response := createNewRequest(http.MethodPut,
			"/api/books/"+isbn, jsonBook, db)
		b, _ := ioutil.ReadAll(response.Body)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json "+
			"content type application/json")
		assertStatus(t, response.Code, http.StatusForbidden, "Should jave status "+
			"code 403: statusForbidden")
		assertError(t, string(b), "Not allowed to change ISBN")
	})

	t.Run("Spamming update which is not allowed ", func(t *testing.T) {
		// Arange
		isbn := "1233211233215"
		want := Book{
			ISBN:  "1233211233215",
			Title: "Star wars phantom menance",
			Author: &Author{
				FirstName: "george",
				LastName:  "lucas"},
			Publisher: "adlibris"}
		dataInfo := &want
		jsonBook, _ := json.Marshal(dataInfo)

		//Update first time
		_ = createNewRequest(http.MethodPut,
			"/api/books/"+isbn, jsonBook, db)

		//Try to update before 10 seconds have passed
		time.Sleep(5 * time.Second)

		//act
		response := createNewRequest(http.MethodPut,
			"/api/books/"+isbn, jsonBook, db)
		b, _ := ioutil.ReadAll(response.Body)

		//assert
		assertContentType(t, response, jsonContentType, "Should have the json"+
			" content type application/json")
		assertStatus(t, response.Code, http.StatusTooEarly, "Should jave status "+
			"code 425: statusToEarly")
		assertError(t, string(b), "Updated a few seconds ago, please wait a "+
			"moment before updating again")
	})
}
