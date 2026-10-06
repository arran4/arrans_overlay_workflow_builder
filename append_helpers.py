with open("inputconfig.go", "a") as f:
    f.write("""
func emptyOrLastBoolPtr(i []string) *bool {
	if len(i) == 0 {
		return nil
	}
	v := strings.ToLower(strings.TrimSpace(i[len(i)-1]))
	b := v == "true" || v == "yes" || v == "1" || v == "on"
	return &b
}

func emptyOrLastPtr(i []string) *string {
	if len(i) == 0 {
		return nil
	}
	v := i[len(i)-1]
	return &v
}
""")
