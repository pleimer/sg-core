package data

import "fmt"

//DataSource indentifies a format of incoming data in the message bus channel.
type DataSource int

//ListAll returns slice of supported data sources in human readable names.
func (src DataSource) ListAll() []string {
	return []string{"generic", "collectd", "ceilometer"}
}

//SetFromString resets value according to given human readable identification. Returns false if invalid identification was given.
func (src *DataSource) SetFromString(name string) bool {
	for index, value := range src.ListAll() {
		if name == value {
			*src = DataSource(index)
			return true
		}
	}
	return false
}

//String returns human readable data type identification.
func (src DataSource) String() string {
	return (src.ListAll())[src]
}

//Prefix returns human readable data type identification.
func (src DataSource) Prefix() string {
	return fmt.Sprintf("%s_*", src.String())
}
