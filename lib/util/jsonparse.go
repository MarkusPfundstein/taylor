package util

import (
	"errors"
	"fmt"
)


func GetBool(cfg map[string]interface{}, key string, def bool) (bool, error) {
	v, in := cfg[key]
	if in == false {
		return def, nil
	}
	casted, ok := v.(bool)
	if ok == false {
		return def, errors.New(fmt.Sprintf("Error casting %s to bool", key))
	}
	return casted, nil
}

func GetString(cfg map[string]interface{}, key string, def string) (string, error) {
	v, in := cfg[key]
	if in == false {
		return def, nil
	}
	casted, ok := v.(string)
	if ok == false {
		return def, errors.New(fmt.Sprintf("Error casting %s to string", key))
	}
	return casted, nil
}

func GetArrayOfStrings(cfg map[string]interface{}, key string, def []string) ([]string, error) {
	v, in := cfg[key]
	if in == false {
		return def, nil
	}

	var tmp []interface{}
	tmp, ok := v.([]interface{})
	if ok == false {
		return def, errors.New(fmt.Sprintf("Error casting %s to []string", key))
	}
	if len(tmp) == 0 {
		return def, nil
	}

	casted := make([]string, len(tmp))
	for i, arg := range tmp {
		casted[i], ok = arg.(string)
		if ok == false {
			return def, errors.New(fmt.Sprintf("Error casting array element %d of %s to string", i, key))
		}
	}
	return casted, nil
}

