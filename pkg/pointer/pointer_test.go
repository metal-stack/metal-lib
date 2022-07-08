package pointer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPointer(t *testing.T) {
	// test for strings
	testString := "test"
	gotString := Pointer(testString)
	if diff := cmp.Diff(gotString, &testString); diff != "" {
		t.Errorf("Pointer() = %s", diff)
	}

	// test for bool
	testBool := true
	gotBool := Pointer(testBool)
	if diff := cmp.Diff(gotBool, &testBool); diff != "" {
		t.Errorf("Pointer() = %s", diff)
	}

	// test for object
	type testStructDef struct{}
	testStruct := testStructDef{}
	getStruct := Pointer(testStruct)
	if diff := cmp.Diff(getStruct, &testStruct); diff != "" {
		t.Errorf("Pointer() = %s", diff)
	}
}

func TestPointerOrDefault(t *testing.T) {
	// test for strings
	testString := "test"
	testStringDefault := "default"
	gotString := PointerOrDefault(testString, testStringDefault)
	if diff := cmp.Diff(gotString, &testString); diff != "" {
		t.Errorf("PointerOrDefault() = %s", diff)
	}
	gotString = PointerOrDefault("", testStringDefault)
	if diff := cmp.Diff(gotString, &testStringDefault); diff != "" {
		t.Errorf("PointerOrDefault() = %s", diff)
	}

	// test for object
	type testStructDef struct{ V string }
	testStruct := testStructDef{V: "test"}
	testStructDefault := testStructDef{V: "default"}
	gotStruct := PointerOrDefault(testStruct, testStructDefault)
	if diff := cmp.Diff(gotStruct, &testStruct); diff != "" {
		t.Errorf("PointerOrDefault() = %s", diff)
	}
	gotStruct = PointerOrDefault(testStructDef{}, testStructDefault)
	if diff := cmp.Diff(gotStruct, &testStructDefault); diff != "" {
		t.Errorf("PointerOrDefault() = %s", diff)
	}
}

func TestDeref(t *testing.T) {
	// test for strings
	testString := "test"
	gotString := Deref(&testString)
	if diff := cmp.Diff(gotString, testString); diff != "" {
		t.Errorf("Deref() = %s", diff)
	}

	// test for bool
	testBool := true
	gotBool := Deref(&testBool)
	if diff := cmp.Diff(gotBool, testBool); diff != "" {
		t.Errorf("Deref() = %s", diff)
	}

	// test for object
	type testStructDef struct{}
	testStruct := testStructDef{}
	getStruct := Deref(&testStruct)
	if diff := cmp.Diff(getStruct, testStruct); diff != "" {
		t.Errorf("Deref() = %s", diff)
	}
}

func TestDerefOrDefault(t *testing.T) {
	// test for strings
	testString := "test"
	var testStringZero string
	testStringDefault := "default"
	gotString := DerefOrDefault(&testString, testStringDefault)
	if diff := cmp.Diff(gotString, testString); diff != "" {
		t.Errorf("DerefOrDefault() = %s", diff)
	}
	gotString = DerefOrDefault(nil, testStringDefault)
	if diff := cmp.Diff(gotString, testStringDefault); diff != "" {
		t.Errorf("DerefOrDefault() = %s", diff)
	}
	gotString = DerefOrDefault(&testStringZero, testStringDefault)
	if diff := cmp.Diff(gotString, testStringDefault); diff != "" {
		t.Errorf("DerefOrDefault() = %s", diff)
	}

	// test for object
	type testStructDef struct{ V string }
	testStruct := testStructDef{V: "test"}
	var testStructZero testStructDef
	testStructDefault := testStructDef{V: "default"}
	gotStruct := DerefOrDefault(&testStruct, testStructDefault)
	if diff := cmp.Diff(gotStruct, testStruct); diff != "" {
		t.Errorf("DerefOrDefault() = %s", diff)
	}
	gotStruct = DerefOrDefault(&testStructDef{}, testStructDefault)
	if diff := cmp.Diff(gotStruct, testStructDefault); diff != "" {
		t.Errorf("DerefOrDefault() = %s", diff)
	}
	gotStruct = DerefOrDefault(&testStructZero, testStructDefault)
	if diff := cmp.Diff(gotStruct, testStructDefault); diff != "" {
		t.Errorf("DerefOrDefault() = %s", diff)
	}
}
