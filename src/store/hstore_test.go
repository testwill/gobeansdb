package store

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var (
	// common
	tBase     = flag.String("base", "/tmp/test_gobeansdb", "base dir of test")
	tNotClear = flag.Bool("notclear", false, "dump meta and key when Testcompatibility")

	// HTree config in TestLoadHints
	tNumbucket = flag.Int("buckets", 16, "#bucket : 1, 16, 256")
	tHeigth    = flag.Int("height", 4, "heigh of HTree (a single root is 1)")
	tLeafSize  = flag.Int("leafsize", 30, "#item of a leaf")
	tPos       = flag.String("pos", "0", "hexString, e.g. ff")

	// profile
	tKeysPerGC = flag.Int("gckeys", 0, "gc per # keys added")
	tPort      = flag.Int("port", 0, "http port for pprof")
	tDoProf    = flag.Bool("prof", false, "do cpu prof for each bench")
	tParallel  = flag.Int("parallel", 0, "do parallel set")

	// input data
	tDataDir  = flag.String("datadir", "", "directory in which to load data files for test")
	tDataFile = flag.String("data", "", "path of datafile to Testcompatibility")

	// verbosity
	tDumpRecord = flag.Bool("dumprecord", false, "dump meta and key when Testcompatibility")
)

var (
	dir           string
	recordPerFile = 6 // even numb
)

func init() {
	os.MkdirAll(*tBase, 0777)
	doProf = *tDoProf
}

func setupTest(casename string, numhome int) {
	initDefaultConfig()
	// dir = time.Now().Format("20060102T030405")
	dir = fmt.Sprintf("%s/%s", *tBase, casename)
	logger.Infof("test in %s", dir)
	os.RemoveAll(dir)
	os.Mkdir(dir, 0777)
	for i := 0; i < numhome; i++ {
		home := fmt.Sprintf("%s/home_%d", dir, i)
		os.Mkdir(home, 0777)
		config.Homes = append(config.Homes, home)
	}
}
func clearTest() {
	if *tNotClear {
		return
	}
	os.RemoveAll(dir)
}

type kvgen struct{}

func (g *kvgen) gen(ki *KeyInfo, i int) (payload *Payload) {
	ki.StringKey = fmt.Sprintf("key_%d", i)
	ki.Key = []byte(ki.StringKey)
	ki.Prepare()
	value := fmt.Sprintf("value_%d", i)
	payload = &Payload{
		Meta: Meta{
			TS:  uint32(i),
			Ver: 1},
		Value: []byte(value),
	}
	return
}

func TestHStoreMem(t *testing.T) {
	testHStore(t, 0)
}

func TestHStoreFlush(t *testing.T) {
	testHStore(t, 1)
}

func TestHStoreRestart(t *testing.T) {
	testHStore(t, 2)
}

func testHStore(t *testing.T, op int) {
	initDefaultConfig()
	gen := kvgen{}

	setupTest(fmt.Sprintf("testHStore_%d", op), 1)
	defer clearTest()
	config.NumBucket = 1
	config.Buckets = make([]int, 16)
	config.Buckets[0] = 1
	config.TreeHeight = 3
	config.Init()

	bucketDir := filepath.Join(config.Homes[0], "0") // TODO: auto create?
	os.Mkdir(bucketDir, 0777)

	store, err := NewHStore()
	if err != nil {
		t.Fatal(err)
	}

	// set

	N := 10
	var ki KeyInfo
	for i := 0; i < N; i++ {
		payload := gen.gen(&ki, i)
		if err := store.Set(&ki, payload); err != nil {
			t.Fatal(err)
		}
	}
	switch op {
	case 1:
		store.flushdatas(true)
	case 2:
		store.Close()
		logger.Infof("closed")
		store, err = NewHStore()

	}

	// get
	for i := 0; i < N; i++ {
		payload := gen.gen(&ki, i)
		payload2, pos, err := store.Get(&ki, false)
		if err != nil {
			t.Fatal(err)
		}
		if payload2 == nil || (string(payload.Value) != string(payload2.Value)) || (pos != Position{0, uint32(PADDING * i)}) {
			t.Fatalf("%d: %#v %#v", i, payload2, pos)
		}
	}
}
