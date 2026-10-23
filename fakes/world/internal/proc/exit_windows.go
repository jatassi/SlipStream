package proc

// waitExit returns once the process pid has exited.
func waitExit(pid int) error {
	w, err := Watch(pid)
	if err != nil {
		return err
	}
	defer w.Close()
	return w.Wait()
}
