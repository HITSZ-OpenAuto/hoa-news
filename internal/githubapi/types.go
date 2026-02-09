package githubapi

type Repo struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

type Commit struct {
	Commit struct {
		Author struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}
