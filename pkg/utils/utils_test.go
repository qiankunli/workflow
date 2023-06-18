package utils

import "testing"

func TestEncodeResourceName(t *testing.T) {
	t.Log(EncodeResourceName("test"))
	t.Log(EncodeResourceName(`#/bin/bash
echo "hello world"`))
}

func TestHashSha256(t *testing.T) {
	t.Log(HashSha256("test"))
	t.Log(HashSha256(`#/bin/bash
echo "hello world"`))
}
