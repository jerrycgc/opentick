package opentick

import (
	"fmt"
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
	"github.com/stretchr/testify/assert"
	"testing"
)

var d = NewTableColDef("Test", Double)

func Test_EncodeTableColDef(t *testing.T) {
	bytes := d.encode()
	d2 := TableColDef{}
	decodeTableColDef(bytes, &d2, schemeVersion)
	assert.Equal(t, d2.Name, d.Name)
	assert.Equal(t, d2.Type, d.Type)
}

func Benchmark_DecodeTableColDef(b *testing.B) {
	b.StopTimer()
	bytes := d.encode()
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		d2 := TableColDef{}
		decodeTableColDef(bytes, &d2, schemeVersion)
	}
}

var cols = []TableColDef{d, d, d}
var tbl = NewTableScheme(cols, []int{2, 1})

func Test_EncodeTableScheme(t *testing.T) {
	bytes := tbl.encode()
	t2 := decodeTableScheme(bytes)
	assert.Equal(t, t2.Cols[2], tbl.Cols[2])
	assert.Equal(t, *t2.Keys[1], *tbl.Keys[1])
}

func Benchmark_DecodeTableScheme(b *testing.B) {
	b.StopTimer()
	bytes := tbl.encode()
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		decodeTableScheme(bytes)
	}
}

func Benchmark_SubspacePack(b *testing.B) {
	b.StopTimer()
	sub := subspace.FromBytes([]byte("test"))
	t := tuple.Tuple{"test", "test", "test"}
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		sub.Pack(t)
	}
}

func Benchmark_SubspaceUnpack(b *testing.B) {
	b.StopTimer()
	sub := subspace.FromBytes([]byte("test"))
	t := tuple.Tuple{"test", "test", "test"}
	p := sub.Pack(t)
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		sub.Unpack(p)
	}
}

func Benchmark_TuplePack(b *testing.B) {
	b.StopTimer()
	t := tuple.Tuple{"test", "test", "test"}
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		t.Pack()
	}
}

func Benchmark_TupleUnpack(b *testing.B) {
	b.StopTimer()
	t := tuple.Tuple{"test", "test", "test"}
	p := t.Pack()
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		tuple.Unpack(p)
	}
}

func concat(a []byte, b ...byte) []byte {
	r := make([]byte, len(a)+len(b))
	copy(r, a)
	copy(r[len(a):], b)
	return r
}

func Benchmark_concat(b *testing.B) {
	b.StopTimer()
	x := []byte("test")
	y := []byte("test")
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		concat(x, y...)
	}
}

func Benchmark_append(b *testing.B) {
	b.StopTimer()
	x := []byte("test")
	y := []byte("test")
	var z []byte
	b.StartTimer()
	for i := 0; i < b.N; i++ { //use b.N for looping
		z = append(x, y...)
	}
	fmt.Print(len(z))
}

func Test_CreateTable(t *testing.T) {
	fdb.MustAPIVersion(FdbVersion)
	var db = fdb.MustOpenDefault()
	sqlCreateTable1 := `
	create table test.test(
		symbol_id bigint,
  	tm timestamp,
		interval int, 
		open double,
		high double,
		low double,
		close double,
		volume double,
		primary key (symbol_id, interval, tm)
	)
  `
	DropDatabase(db, "test")
	ast, _ := Parse(sqlCreateTable1)
	err := CreateTable(db, "", ast.Create.Table)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "Database test does not exist")
	CreateDatabase(db, "test")

	ast1, _ := Parse("create table test.test(x int, y int, x int)")
	err = CreateTable(db, "", ast1.Create.Table)
	assert.Equal(t, err.Error(), "Multiple definition of identifier x")

	ast2, _ := Parse("create table test.test(x int, y int, primary key (x, x))")
	err = CreateTable(db, "", ast2.Create.Table)
	assert.Equal(t, err.Error(), "Duplicate definition x referenced in PRIMARY KEY")

	ast3, _ := Parse("create table test.test(x int, y int)")
	err = CreateTable(db, "", ast3.Create.Table)
	assert.Equal(t, err.Error(), "PRIMARY KEY not declared")

	ast4, _ := Parse("create table test.test(x int, y int, primary key(x, z))")
	err = CreateTable(db, "", ast4.Create.Table)
	assert.Equal(t, err.Error(), "Unknown definition z referenced in PRIMARY KEY")

	err = CreateTable(db, "", ast.Create.Table)
	assert.Equal(t, err, nil)
	tbl, err1 := GetTableScheme(db, "test", "test")
	assert.Equal(t, err1, nil)
	assert.Equal(t, tbl.Cols[7].Name, "volume")
	assert.Equal(t, tbl.Keys[1].Name, "interval")
	assert.Equal(t, tbl.NameMap["symbol_id"].PosCol, uint32(0))
	assert.Equal(t, tbl.NameMap["tm"].PosCol, uint32(1))
	assert.Equal(t, tbl.NameMap["tm"].Pos, uint32(2))
	assert.Equal(t, tbl.NameMap["close"].PosCol, uint32(6))
	assert.Equal(t, tbl.NameMap["close"].Pos, uint32(3))
	dir, _ := directory.Open(db, []string{"db", "test", "test"}, nil)
	dir2, _ := directory.Open(db, []string{"db", "test", "test", "scheme"}, nil)
	assert.Equal(t, len(dir.Bytes()), len(dir2.Bytes()))
	assert.Equal(t, string(tbl.Dir.Bytes()), string(dir.Bytes()))
}