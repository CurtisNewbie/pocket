package main

func main() {
	if err := NewApp().Run(); err != nil {
		panic(err)
	}
}
