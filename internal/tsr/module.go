package tsr

// Module represents an imported TSR module
type Module struct {
        Name    string
        Members map[string]*Value
}
