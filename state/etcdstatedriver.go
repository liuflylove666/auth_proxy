package state

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	//"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	//"github.com/coreos/etcd/client"
	client "go.etcd.io/etcd/clientv3"
	"golang.org/x/net/context"

	auth_errors "github.com/contiv/auth_proxy/common/errors"
	"github.com/contiv/auth_proxy/common/types"
)

const (

	// timeout used for etcd client calls
	ctxTimeout = 20 * time.Second

	// max times to retry in case of failure
	maxEtcdRetries = 10
)

// EtcdStateDriver implements the StateDriver interface for an etcd-based
// distributed key-value store that is used to store any state information
// needed by auth proxy
type EtcdStateDriver struct {

	// Client to access etcd
	Client *client.Client

	// KeysAPI is used to interact with etcd's key-value
	// API over HTTP
	// KeysAPI client.KeysAPI
}

//
// Init initializes the state driver with needed config
//
// Parameters:
//   config: configuration parameters to create etcd client
//
// Return values:
//   error:  error when creating an etcd client
//
func (d *EtcdStateDriver) Init(config *types.KVStoreConfig) error {
	var err error
	var endpoint *url.URL

	if config == nil || len(config.StoreURL) == 0 {
		return errors.New("Invalid etcd config")
	}

	for  _,dburl :=  range config.StoreURL {

		endpoint, err = url.Parse(dburl)
		if err != nil {
			return err
		}

		if endpoint.Scheme == "etcd" {
			endpoint.Scheme = "http"
		} else if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
			return fmt.Errorf("invalid etcd URL scheme %q", endpoint.Scheme)
		}
	}


	etcdConfig := client.Config{
		// Endpoints: []string{endpoint.String()},
		Endpoints: config.StoreURL,
	}

	// create etcd client
	d.Client, err = client.New(etcdConfig)
	if err != nil {
		log.Fatalf("failed to create etcd client, err: %v", err)
	}

	// create keys api
	// d.KeysAPI = client.NewKeysAPI(d.Client)

	for _, dir := range types.DatastoreDirectories {
		// etcd paths begin with a slash
		d.Mkdir("/" + dir)
	}

	return nil
}

// Deinit is currently a no-op
func (d *EtcdStateDriver) Deinit() {}

// Mkdir creates a directory.  If it already exists, this is a no-op.
//
// Parameters:
//   key: target directory path (must begin with a slash)
//
// Return values:
//   error: Error encountered when creating the directory
//   nil:   successfully created directory
//
func (d *EtcdStateDriver) Mkdir(key string) error {
	// ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	// defer cancel()

	// // sanity test
	// if !strings.HasPrefix(key, "/") {
	// 	return fmt.Errorf(
	// 		"etcd keys must begin with a slash (got '%s')",
	// 		key,
	// 	)
	// }

	// for i := 0; ; i++ {
	// 	_, err := d.KeysAPI.Set(ctx, key, "", &client.SetOptions{Dir: true})
	// 	if err == nil {
	// 		return nil
	// 	}

	// 	// Retry few times if cluster is unavailable
	// 	if err.Error() == client.ErrClusterUnavailable.Error() {
	// 		if i < maxEtcdRetries {
	// 			// Retry after a delay
	// 			time.Sleep(time.Second)
	// 			continue
	// 		}
	// 	}

	// 	return err
	// }
	// etcd 3 does not have directories
	return nil
}

//
// Write state (consisting of a key-value pair) to the etcd
// KV store
//
// Parameters:
//   key:    key to be stored
//   value:  value to be stored
//
// Return values:
//   error: Error when writing to the KeysAPI of etcd client
//          nil if successful
//
func (d *EtcdStateDriver) Write(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	_, err := d.Client.KV.Put(ctx, key, string(value[:]))
	if err != nil {
		// Retry few times if cluster is unavailable
		if err == client.ErrNoAvailableEndpoints {
			for i := 0; i < maxEtcdRetries; i++ {
				_, err = d.Client.KV.Put(ctx, key, string(value[:]))
				if err == nil {
					break
				}

				// Retry after a delay
				time.Sleep(time.Second)
			}
		}
	}

	return err
}

//
// Read returns state for a key
//
// Parameters:
//   key:    key for which value is to be retrieved
//
// Return values:
//   []byte: value associated with the given key
//   error: Error when writing to the KeysAPI of etcd client
//          nil if successful
//
func (d *EtcdStateDriver) Read(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	var err error
	var resp *client.GetResponse

	// i <= maxEtcdRetries to ensure that the initial `GET` call is also incorporated along with retries
	for i := 0; i <= maxEtcdRetries; i++ {
		resp, err = d.Client.KV.Get(ctx, key)
		if err != nil {
			if err == client.ErrNoAvailableEndpoints {
				// retry after a delay
				time.Sleep(time.Second)
			}
			continue
		}
		if resp.Count == 0 {
			return nil, auth_errors.ErrKeyNotFound
		}

		// on successful read
		return resp.Kvs[0].Value, nil

	}

	return []byte{}, err
}

//
// ReadAll returns all values for a key
//
// Parameters:
//   key:    key for which all values are to be retrieved
//
// Return values:
//   [][]byte: slice of values associated with the given key
//   error:    Error when writing to the KeysAPI of etcd client
//             nil if successful
//
func (d *EtcdStateDriver) ReadAll(baseKey string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	var err error
	var resp *client.GetResponse

	// i <= maxEtcdRetries to ensure that the initial `GET` call is also incorporated along with retries
	for i := 0; i <= maxEtcdRetries; i++ {
		resp, err = d.Client.KV.Get(ctx, baseKey, client.WithPrefix())
                log.Debugf("GetResponse1111:%+v",resp)
		if err != nil {
			if err == client.ErrNoAvailableEndpoints {
				// retry after a delay
				time.Sleep(time.Second)

			}
			continue
		}
		if resp.Count == 0 {
			return nil, auth_errors.ErrKeyNotFound
		}

		// on successful read
		values := [][]byte{}
		for _, node := range resp.Kvs {
			values = append(values, node.Value)
		}
	        return values, nil	
                log.Debugf("aaaaffffvalues:%+v", values)
	}
	return [][]byte{}, err
}

//
// channelEtcdEvents
//
// Parameters:
//   watcher:        Any struct that implements the Watcher interface provided
//                   by the etcd client
//   chValueChanges: Channel of type [2][]byte used to communicate the value changes
//                   for a key in the KV store
//
// func (d *EtcdStateDriver) channelEtcdEvents(watcher client.Watcher, chValueChanges chan [2][]byte) {
func (d *EtcdStateDriver) channelEtcdEvents(watcher client.WatchChan, rsps chan [2][]byte) {
        for resp := range watcher {

                for _, ev := range resp.Events {
                        //                      fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)

                        rsp := [2][]byte{nil, nil}
                        eventStr := "create"
                        if string(ev.Kv.Value) != "" {
                                rsp[0] = ev.Kv.Value
                        }

                        if ev.PrevKv != nil && string(ev.PrevKv.Value) != "" {
                                rsp[1] = ev.PrevKv.Value
                                if string(ev.Kv.Value) != "" {
                                        eventStr = "modify"
                                } else {
                                        eventStr = "delete"
                                }
                        }

                        log.Debugf("Received %q for key: %s", eventStr, ev.Kv.Key)
                        //channel the translated response
                        rsps <- rsp
                }
        }
}

//
// WatchAll watches value changes for a key in etcd
//
// Parameters:
//   baseKey:        key for which changes are to be watched
//   chValueChanges: channel for communicating the changes in
//                   the values for a key from this method
//
// Return values:
//   error: Any error when watching for a state transition
//          for a key
//          nil if successful
//
// func (d *EtcdStateDriver) WatchAll(baseKey string, chValueChanges chan [2][]byte) error {
func (d *EtcdStateDriver) WatchAll(baseKey string, rsps chan [2][]byte) error {

	// watcher := d.KeysAPI.Watcher(baseKey, &client.WatcherOptions{Recursive: true})
	// if watcher == nil {
	// 	log.Errorf("etcd watch failed")
	// 	return errors.New("etcd watch failed")
	// }
	watcher := d.Client.Watch(context.Background(), baseKey, client.WithPrefix())
	go d.channelEtcdEvents(watcher, rsps)

	return nil
}

//
// Clear removes a key from etcd
//
// Parameters:
//   key: key to be removed
//
// Return value:
//   error: Error returned by etcd client when deleting a key
//
func (d *EtcdStateDriver) Clear(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	var resp *client.DeleteResponse

	resp, err := d.Client.KV.Delete(ctx, key)
	if resp.Deleted == 0 {
		return auth_errors.ErrKeyNotFound
	}

	return err
}

//
// ClearState removes a key from etcd
//
// Parameters:
//   key: key to be removed
//
// Return value:
//   error: Error returned by etcd client when deleting a key
//
func (d *EtcdStateDriver) ClearState(key string) error {
	return d.Clear(key)
}

//
// ReadState reads a key's value into a types.State struct using
// the provided unmarshaling function.
//
// Parameters:
//   key:       key whose value is to be retrieved
//   value:     value of the key as types.State
//   unmarshal: function to be used for unmarshaling the (byte
//              slice) value into types.State struct
//
// Return value:
//   error: Error returned by etcd client when reading key's value
//          or error in unmarshaling key's value
//
func (d *EtcdStateDriver) ReadState(key string, value types.State,
	unmarshal func([]byte, interface{}) error) error {

	encodedState, err := d.Read(key)
	if err != nil {
		return err
	}

	return unmarshal(encodedState, value)
}

//
// readAllStateCommon reads and unmarshals (given a function) all state into a
// slice of type types.State
//
// Parameters:
//   d:          StateDriver abstracting etcd or consul KV store
//   baseKey:    key whose state is to be read from KV store
//   sType:      State
//   unmarshal:  Unmarshal function to convert key's value from a byte slice
//               to a struct of type types.State
//
// Return value:
//   []types.State: slice of states
//   error:         Error returned when reading the key and unmarshaling it
//                  nil if successful
//
func readAllStateCommon(d types.StateDriver, baseKey string, sType types.State,
	unmarshal func([]byte, interface{}) error) ([]types.State, error) {

	stateType := reflect.TypeOf(sType)
	sliceType := reflect.SliceOf(stateType)
	values := reflect.MakeSlice(sliceType, 0, 1)

        log.Debugf("1222stateType:%+v", stateType)
        log.Debugf("1222sliceType:%+v", sliceType)
        log.Debugf("1222values:%+v", values)
        log.Debugf("1222baseKey:%+v", baseKey)
	byteValues, err := d.ReadAll(baseKey)
         
        log.Debugf("1222byteValues:%+v", byteValues)
	if err != nil {
		return nil, err
	}
	for _, byteValue := range byteValues {
		value := reflect.New(stateType)
		err = unmarshal(byteValue, value.Interface())
		if err != nil {
			return nil, err
		}
		values = reflect.Append(values, value.Elem())
	}

	stateValues := []types.State{}
	for i := 0; i < values.Len(); i++ {
		// sanity checks
		if !values.Index(i).Elem().FieldByName("CommonState").IsValid() {
			return nil, auth_errors.ErrCommonStateFieldsMissing
		}

		//the following works as every types.State is expected to embed core.CommonState struct
		values.Index(i).Elem().FieldByName("CommonState").FieldByName("StateDriver").Set(reflect.ValueOf(d))
		stateValue := values.Index(i).Interface().(types.State)
		stateValues = append(stateValues, stateValue)
	}
	return stateValues, nil
}

//
// ReadAllState returns all state for a key
//
// Parameters:
//   baseKey:    key whose values are to be read
//   sType:      types.State struct into which values are to be
//               unmarshaled
//   unmarshal:  function that is used to convert key's values to
//               values of type types.State
//
// Return values:
//   []types.State: slice of states for the given key
//   error:         Any error returned by readAllStateCommon
//                  nil if successful
//
func (d *EtcdStateDriver) ReadAllState(baseKey string, sType types.State,
	unmarshal func([]byte, interface{}) error) ([]types.State, error) {
	return readAllStateCommon(d, baseKey, sType, unmarshal)
}

//
// channelStateEvents watches for updates (create, modify, delete) to a state of
// specified type and unmarshals (given a function) all changes and puts them on
// a channel of types.WatchState objects.
//
// Parameters:
//    d:              StateDriver that abstracts access to etcd or consul
//    sType:          types.State
//    unmarshal:      function used to unmarshall byte slice values into
//                    type types.State
//    chValueChanges: channel of [2][]byte via which this method
//                    returns any value changes that were observed in the KV store
//    chStateChanges: channel of type types.WatchState via which this method
//                    returns any state changes that were observed in the KV store
//    chErr:          channel of type error via which this method returns
//                    any errors encountered
//
func channelStateEvents(d types.StateDriver, sType types.State,
	unmarshal func([]byte, interface{}) error,
	chValueChanges chan [2][]byte,
	chStateChanges chan types.WatchState,
	chErr chan error) {

	for {
		// block on change notifications
		byteRsp := <-chValueChanges

		stateChange := types.WatchState{Curr: nil, Prev: nil}
		for i := 0; i < 2; i++ {
			if byteRsp[i] == nil {
				continue
			}
			stateType := reflect.TypeOf(sType)
			value := reflect.New(stateType)
			err := unmarshal(byteRsp[i], value.Interface())
			if err != nil {
				chErr <- err
				return
			}
			if !value.Elem().Elem().FieldByName("CommonState").IsValid() {
				chErr <- auth_errors.ErrCommonStateFieldsMissing
				return
			}

			//the following works as every types.State is expected to embed core.CommonState struct
			value.Elem().Elem().FieldByName("CommonState").FieldByName("StateDriver").Set(reflect.ValueOf(d))
			switch i {
			case 0:
				stateChange.Curr = value.Elem().Interface().(types.State)
			case 1:
				stateChange.Prev = value.Elem().Interface().(types.State)
			}
		}

		// send state changes for the key to a channel
		chStateChanges <- stateChange
	}
}

//
// WatchAllState watches all state from the baseKey
//
// Parameters:
//    baseKey:        key to be watched
//    sType:          types.State struct to convert values to
//    unmarshal:      function used to convert values to types.State
//    chStateChanges: channel of types.WatchState
//
// Return values:
//    error: Any error when watching all state
//
func (d *EtcdStateDriver) WatchAllState(baseKey string, sType types.State,
	unmarshal func([]byte, interface{}) error, chStateChanges chan types.WatchState) error {

	// channel that will be used to communicate value changes
	// from the WatchAll function
	chValueChanges := make(chan [2][]byte, 1)

	// channel that will be used to communicate errors
	// from the channelStateEvents method
	chErr := make(chan error, 1)

	go channelStateEvents(d, sType, unmarshal, chValueChanges, chStateChanges, chErr)

	err := d.WatchAll(baseKey, chValueChanges)
	if err != nil {
		return err
	}

	err = <-chErr
	return err
}

//
// WriteState writes a value of types.State for a key in the KV store
//
// Parameters:
//   key:   key to be stored in the KV store
//   value: value as types.State
//   marshal: function to be used to convert types.State to a form
//            that can be stored in the KV store
//
// Return values:
//   error: Error while marshaling or writing a key-value pair
//          to the KV store
//
func (d *EtcdStateDriver) WriteState(key string, value types.State,
	marshal func(interface{}) ([]byte, error)) error {
	log.Debugf("aaaaafffssss")
	encodedState, err := marshal(value)
	if err != nil {
		return err
	}

	return d.Write(key, encodedState)
}
