// Package costs owns exact decimal arithmetic. JSON uses decimal strings.
package costs

import (
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"strings"
)

var ErrInvalid = errors.New("invalid cost input")
var decimal = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`)

type Money string

func (m *Money) UnmarshalJSON(b []byte) error {
	var s string
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
	} else {
		s = string(b)
	}
	if _, err := Parse(s); err != nil {
		return err
	}
	*m = Money(s)
	return nil
}
func Parse(s string) (*big.Rat, error) {
	if len(s) > 80 || !decimal.MatchString(s) {
		return nil, ErrInvalid
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, ErrInvalid
	}
	return r, nil
}

// Round applies half-away-from-zero once at the posting line, not per factor.
func Round(r *big.Rat) Money {
	n := new(big.Int).Mul(r.Num(), big.NewInt(100))
	sign := n.Sign()
	n.Abs(n)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, r.Denom(), rem)
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	s := q.String()
	for len(s) < 3 {
		s = "0" + s
	}
	s = s[:len(s)-2] + "." + s[len(s)-2:]
	if sign < 0 && q.Sign() != 0 {
		s = "-" + s
	}
	return Money(s)
}
func Normalize(m Money) (Money, error) {
	r, err := Parse(string(m))
	if err != nil {
		return "", err
	}
	out := Round(r)
	if len(strings.Split(strings.TrimPrefix(string(out), "-"), ".")[0]) > 16 {
		return "", ErrInvalid
	}
	return out, nil
}
func Multiply(quantity, rate string) (Money, error) {
	q, err := Parse(quantity)
	if err != nil {
		return "", err
	}
	r, err := Parse(rate)
	if err != nil {
		return "", err
	}
	return Normalize(Round(new(big.Rat).Mul(q, r)))
}
func Add(a, b Money) Money {
	x, _ := Parse(string(a))
	y, _ := Parse(string(b))
	return Round(new(big.Rat).Add(x, y))
}
func Sub(a, b Money) Money {
	x, _ := Parse(string(a))
	y, _ := Parse(string(b))
	return Round(new(big.Rat).Sub(x, y))
}
func Compare(a, b Money) int { x, _ := Parse(string(a)); y, _ := Parse(string(b)); return x.Cmp(y) }
func Tax(amount Money, basis, rate string) (Money, Money, error) {
	a, e := Parse(string(amount))
	if e != nil {
		return "", "", e
	}
	r, e := Parse(rate)
	if e != nil || r.Sign() < 0 || r.Cmp(big.NewRat(1, 1)) > 0 {
		return "", "", ErrInvalid
	}
	if basis == "inc_tax" {
		net := new(big.Rat).Quo(a, new(big.Rat).Add(big.NewRat(1, 1), r))
		n := Round(net)
		return n, Sub(amount, n), nil
	}
	if basis != "ex_tax" {
		return "", "", ErrInvalid
	}
	return amount, Round(new(big.Rat).Mul(a, r)), nil
}
