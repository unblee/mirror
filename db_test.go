package main

type TestDB struct {
	desiredData string
	defaultData string
}

func newTestDB(desiredData, defaultData string) DB {
	return &TestDB{
		desiredData: desiredData,
		defaultData: defaultData,
	}
}

func (d *TestDB) get(field string) (string, error) {
	if field == "not.exist" {
		return d.defaultData, nil
	}
	return d.desiredData, nil
}
func (d *TestDB) close() error {
	return nil
}
