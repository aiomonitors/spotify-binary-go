// This example demonstrates how to authenticate with Spotify.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//       - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/zmb3/spotify"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8080/callback"

var html = `
<br/>
<a href="/player/play">Play</a><br/>
<a href="/player/pause">Pause</a><br/>
<a href="/player/next">Next track</a><br/>
<a href="/player/previous">Previous Track</a><br/>
<a href="/player/shuffle">Shuffle</a><br/>
<a href="/binary">Binary Playlist Generator (Return to terminal)</a><br/>
`

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState, spotify.ScopePlaylistModifyPublic)
	ch    = make(chan *spotify.Client)
	state = "slsadlkad"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	auth.SetAuthInfo("e08f7f5bb7fe4feabb37cae240c9c4be", "41c5a6cc79764012ad82708dc5432080")

	// We'll want these variables sooner rather than later
	var client *spotify.Client
	var playerState *spotify.PlayerState

	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		//Scans for string
		color.Blue("Please enter a string between length 1 and 7, no spaces")
		var input string
		fmt.Scanln(&input)

		if len(input) > 7 {
			panic(errors.New("Input too long"))
		}

		binaryString := binary(input)
		screenPrint := fmt.Sprintf("Binary String:%s", binaryString)
		color.Yellow(screenPrint)

		color.Yellow("Compiling trackss")

		var limit int
		limit = 50
		offset := 0
		zeroTracks := []spotify.FullTrack{}
		oneTracks := []spotify.FullTrack{}
		for i := 1; i <= 10; i++ {
			searchOptions := spotify.Options{Limit: &limit, Offset: &offset}
			query, qerr := client.SearchOpt("zero", spotify.SearchTypeTrack, &searchOptions)
			if qerr != nil {
				panic(qerr)
			}
			for _, track := range *&query.Tracks.Tracks {
				if strings.HasPrefix(track.Name, "Zero") {
					zeroTracks = append(zeroTracks, track)
				}
			}

			offset += 50
		}
		offset = 0
		for i := 1; i <= 10; i++ {
			searchOptions := spotify.Options{Limit: &limit, Offset: &offset}
			query, qerr := client.SearchOpt("one", spotify.SearchTypeTrack, &searchOptions)
			if qerr != nil {
				panic(qerr)
			}
			for _, track := range *&query.Tracks.Tracks {
				if strings.HasPrefix(track.Name, "One") {
					oneTracks = append(oneTracks, track)
				}
			}

			offset += 50
		}
		color.Yellow("Compiled tracks")

		fmt.Println(len(oneTracks))
		fmt.Println(len(zeroTracks))

		color.Yellow("Creating playlist")
		user, _ := client.CurrentUser()
		playlist, createErr := client.CreatePlaylistForUser(user.ID, "Binary", "To get the word in binary, look at the song names, if it starts with Zero, add 0 to your binary, and if it starts with One add 1 to your binary. The song space signals a space. Then use www.rapidtables.com/convert/number/binary-to-ascii3.html to convert it to string. Created by @aiomonitors on GitHub", true)
		if createErr != nil {
			panic(createErr)
		}
		color.Green("Playlist created")
		fmt.Println(*&playlist.URI)
		fmt.Println()

		color.Yellow("Compiling IDs to add")
		ids := []spotify.ID{}
		for _, c := range binaryString {
			var randInt int
			switch s := string(c); s {
			case "0":
				randInt = rand.Intn(len(zeroTracks))
				ids = append(ids, zeroTracks[randInt].ID)
			case "1":
				randInt = rand.Intn(len(oneTracks))
				ids = append(ids, oneTracks[randInt].ID)
			case " ":
				ids = append(ids, "6vqGHxoRfKJvD9jWhmuCyD")
			}
		}
		color.Green("Compiled track IDS")
		fmt.Println(ids)

		color.Yellow("Adding tracks to playlist")
		_, addErr := client.AddTracksToPlaylist(playlist.ID, ids...)
		if addErr != nil {
			panic(addErr)
		}
		color.Green("Songs added!")

		color.Green("https://open.spotify.com/playlist/%s", strings.Split(string(*&playlist.URI), ":")[2])
		fmt.Fprintf(w, screenPrint+html)
	})

	http.HandleFunc("/player/", func(w http.ResponseWriter, r *http.Request) {
		action := strings.TrimPrefix(r.URL.Path, "/player/")
		fmt.Println("Got request for:", action)
		var err error
		switch action {
		case "play":
			err = client.Play()
		case "pause":
			err = client.Pause()
		case "next":
			err = client.Next()
		case "previous":
			err = client.Previous()
		case "shuffle":
			playerState.ShuffleState = !playerState.ShuffleState
			err = client.Shuffle(playerState.ShuffleState)
		}
		if err != nil {
			log.Print(err)
		}
		currentPlaying, err := client.PlayerCurrentlyPlaying()
		if err != nil {
			log.Print(err)
		}
		currentTrack := currentPlaying.Item.Name
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, fmt.Sprintf("%s", currentTrack)+html)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})

	go func() {
		url := auth.AuthURL(state)
		fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

		// wait for auth to complete
		client = <-ch

		// use the client to make calls that require authorization
		user, err := client.CurrentUser()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("You are logged in as:", user.ID)

		playerState, err = client.PlayerState()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
	}()

	http.ListenAndServe(":8080", nil)

}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	auth.SetAuthInfo("e08f7f5bb7fe4feabb37cae240c9c4be", "41c5a6cc79764012ad82708dc5432080")
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Login Completed!"+html)
	ch <- &client
}

func binary(s string) string {
	res := ""
	for _, c := range s {
		res = fmt.Sprintf("%s%.8b ", res, c)
	}
	return res
}
