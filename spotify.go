package main

import (
    "fmt"
    "log"
    "net/http"
    "strconv"

    "github.com/zmb3/spotify"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8080/callback"

var html = `
<form action="/library">
    <input type="submit" value="View Library" />
</form>
`

var (
    auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState, spotify.ScopeUserLibraryRead)
    ch    = make(chan *spotify.Client)
    state = "abc123"
)

func main() {
    // We'll want these variables sooner rather than later
    var client *spotify.Client
    var playerState *spotify.PlayerState

    http.HandleFunc("/callback", completeAuth)

    http.HandleFunc("/library", func(w http.ResponseWriter, r *http.Request) {
        log.Println("Got request for:", r.URL.String())
        var err error

        //********TRACK INFO********//
        // TODO remove the /player page and move this stuff somewhere else without breaking everything

        var info *spotify.SavedTrackPage
        var tracks []spotify.SavedTrack
        var limit = 50
        var offset = 0
        var songs = `<table><tr><th align='left'>Song</th><th>Duration</th></tr>`
        var numSongs = 0
        var totalDuration = 0.0

        for {
            opt := &spotify.Options {Limit: &limit, Offset: &offset}
            info, err = client.CurrentUsersTracksOpt(opt)
            if err != nil {
                log.Print(err)
            }
            if len(info.Tracks) == 0 {
                break
            }
            tracks = append(tracks, info.Tracks...)
            offset += 50
        }

        for _, song := range tracks {
            songs += "<tr><td>" + song.Name + "</td><td>" + MillisecondsToTime(int(song.Duration)) + "</td></tr>"
            // TODO convert time to minutes, hours, and days
            totalDuration += float64(song.Duration)
            numSongs++
        }
        songs = fmt.Sprintf("<table><tr><th>Number of Songs</th><td>%d</td></table>%s<tr><td>Total Duration</td><td>%s</td></table>", numSongs, songs, MillisecondsToTime(int(totalDuration)))

        //******END TRACK INFO******//
        
        w.Header().Set("Content-Type", "text/html")
        fmt.Fprint(w, html, songs)
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
    tok, err := auth.Token(state, r)
    if err != nil {
        http.Error(w, "Couldn't get token", http.StatusForbidden)
        log.Fatal(err)
    }
    if st := r.FormValue("state"); st != state {
        http.NotFound(w, r)
        log.Fatalf("State mismatch: %s != %s\n", st, state)
    }
    // use the token to get an authenticated client
    client := auth.NewClient(tok)
    w.Header().Set("Content-Type", "text/html")
    fmt.Fprintf(w, "Login Completed!"+html)
    ch <- &client
}

func MillisecondsToTime(ms int) string {
    seconds := int((float64(ms) / 1000.0) + 0.5)
    time := strconv.Itoa(seconds / 60) + ":" + fmt.Sprintf("%02d", seconds % 60)
    return time
}
