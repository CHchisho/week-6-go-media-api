package main

import "testing"

func TestCleanMovie(t *testing.T) {
	movie := movieInput{
		Title:       " Inception ",
		Director:    "Christopher Nolan",
		Genre:       "Sci-Fi",
		Year:        2010,
		Rating:      8.8,
		Description: "A mind-bending thriller about dreams within dreams.",
	}
	if err := cleanMovie(&movie); err != nil {
		t.Fatal(err)
	}
	if movie.Title != "Inception" {
		t.Fatalf("title = %q", movie.Title)
	}

	movie.Genre = "Romance"
	if err := cleanMovie(&movie); err == nil {
		t.Fatal("expected an error for an unknown genre")
	}
}

func TestOrderBy(t *testing.T) {
	got, err := orderBy("title", "asc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "title ASC" {
		t.Fatalf("got %q", got)
	}

	if _, err := orderBy("title; drop table movies", "desc"); err == nil {
		t.Fatal("expected an error for an unknown sort field")
	}
}

func TestMovieFilters(t *testing.T) {
	where, args := movieFilters("Drama", "100%", 1994)
	if where != " WHERE genre = ? AND release_year = ? AND director LIKE ? ESCAPE '\\\\'" {
		t.Fatalf("where = %q", where)
	}
	if len(args) != 3 || args[2] != `%100\%%` {
		t.Fatalf("args = %#v", args)
	}
}
