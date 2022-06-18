package bot

type Queue struct {
	songs     []Song
	current   *Song
	isPlaying bool
}

func NewQueue() *Queue {
	return &Queue{
		songs: make([]Song, 0),
	}
}

func (q *Queue) Add(song Song) {
	q.songs = append(q.songs, song)
}

func (q *Queue) Get() []Song {
	return q.songs
}

func (q *Queue) Set(songs []Song) {
	q.songs = songs
}

// HasNext checks out if there are any songs after current
func (q *Queue) HasNext() bool {
	return len(q.songs) > 0
}

// Next extract first song and make it current if there's any songs in queue
func (q *Queue) Next() Song {
	if q.HasNext() {
		song := q.songs[0]
		q.songs = q.songs[1:]
		q.current = &song
		return song
	}

	return Song{}
}

// Clear makes empty the queue
func (q *Queue) Clear() {
	q.songs = make([]Song, 0)
	q.current = nil
	q.isPlaying = false
}

// Current return song that playing right now
func (q *Queue) Current() *Song {
	return q.current
}

// Pause stops song until resume
func (q *Queue) Pause() {
	q.isPlaying = false
}

// Play starts to play songs from the queue
func (q *Queue) Play(s *Session, writeChannel func(string)) {
	q.isPlaying = true

	for q.HasNext() && q.isPlaying {
		song := q.Next()
		writeChannel("**Now playing:** " + song.Title + ".")
		s.Play(song)
	}

	if !q.isPlaying {
		writeChannel("Stopped playing.")
	} else {
		writeChannel("Finished queue.")
	}
}
