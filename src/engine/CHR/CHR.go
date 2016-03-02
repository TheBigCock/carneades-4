// Copyright © 2016 The Carneades Authors
// This Source Code Form is subject to the terms of the
// Mozilla Public License, v. 2.0. If a copy of the MPL
// was not distributed with this file, You can obtain one
// at http://mozilla.org/MPL/2.0/.

// Constraint Handling Rules

package chr

import (
	. "github.com/carneades/carneades-4/src/engine/terms"
	// "fmt"
	"math/big"
	// "strconv"
	// "strings"
)

var QueryVars Vars

var QuerySore List

var CHRstore List

var BuiltInStore List

type history [][]*big.Int

// var History []idSequence

var CurVarCounter *big.Int

type cList []Compound

type chrRule struct {
	name     string
	id       int
	his      history
	delHead  cList // removed constraints
	keepHead cList // kept constraint
	guard    cList // built-in constraint
	body     List  // add CHR and built-in constraint
}

var CHRruleStore []*chrRule

func CHRsolver() {
	for ruleFound := true; ruleFound; {
		ruleFound = false
		for _, rule := range CHRruleStore {
			if ruleFired(rule) {
				ruleFound = true
				break
			}
		}
	}
}

func ruleFired(rule *chrRule) (ok bool) {
	headList := rule.delHead
	len_head := len(headList)
	if len_head != 0 {
		ok = unifyDelHead(rule, headList, 0, len_head, nil)
		return ok
	}

	headList = rule.keepHead
	len_head = len(headList)
	if len_head == 0 {
		return false
	}

	ok = unifyKeepHead(rule, []*big.Int{}, headList, 0, len_head, nil)
	return ok
}

func attributedTerm(t Compound) cList {
	return cList{}
}

func unifyDelHead(r *chrRule, headList cList, it int, nt int, env Bindings) (ok bool) {
	var env2 Bindings
	head := headList[it]
	chrList := attributedTerm(head)
	len_chr := len(chrList)
	if len_chr != 0 {
		for ok, ic := false, 0; !ok && ic < len_chr; ic++ {
			chr := chrList[ic]

			env2, ok = mDelUnify(r.id, head, chr, env) // mark chr and Unify, if fail unmark chr
			if ok {
				if it+1 < nt {
					ok = unifyDelHead(r, headList, it+1, nt, env2)
					if ok {
						return ok
					}
				} else {
					// the last delHead-match was OK
					headList = r.keepHead
					nt = len(headList)
					if nt != 0 {
						ok = unifyKeepHead(r, nil, headList, 0, nt, env2)
						if ok {
							return ok
						}
					} else {
						// only delHead
						ok := checkGuards(r, env2)
						if ok {
							return ok
						}
					}
				} // if it+1 < nt
			}
			// mUnify was OK, but rule does not fire OR mUnify was not OK
			// env is the currend environment
			// try the next constrain for the constrain store
		}
		// no constrain from the constraint store match head
	}
	return false
}

func mDelUnify(id int, head, chr Compound, env Bindings) (env2 Bindings, ok bool) {
	// mark and unmark chr
	return Unify(head, chr, env)
}

func mKeepUnify(id int, head, chr Compound, env Bindings) (env2 Bindings, ok bool) {
	// mark and unmark chr
	return Unify(head, chr, env)
}

func unifyKeepHead(r *chrRule, his []*big.Int, headList cList, it int, nt int, env Bindings) (ok bool) {
	var env2 Bindings
	head := headList[it]
	chrList := attributedTerm(head)
	len_chr := len(chrList)
	if len_chr != 0 {
		for ok, ic := false, 0; !ok && ic < len_chr; ic++ {
			chr := chrList[ic]

			env2, ok = mKeepUnify(r.id, head, chr, env) // mark chr and Unify, if fail unmark chr
			if ok {
				if it+1 < nt {
					if his == nil {
						// rule with delHead
						ok = unifyKeepHead(r, nil, headList, it+1, nt, env2)
					} else {
						ok = unifyKeepHead(r, append(his, chr.Id), headList, it+1, nt, env2)
					}

					if ok {
						return ok
					}
				} else {
					// the last keepHead-match was OK
					// check history
					if his == nil || pCHRsNotInHistory(append(his, chr.Id), r.his) {

						ok := checkGuards(r, env2)
						if ok {
							return ok
						}

					}

				} // if it+1 < nt

			}
			// mUnify was OK, but rule does not fire OR mUnify was not OK
			// env is the currend environment
			// try the next constrain of the constrain store
		}
		// no constrain from the constraint store match head
	}
	return false
}

func pCHRsNotInHistory(chrs []*big.Int, his history) (ok bool) {
	return true
}

func checkGuards(r *chrRule, env Bindings) (ok bool) {
	for _, g := range r.guard {
		env2, ok := checkGuard(g, env)
		if !ok {
			return false
		}
		env = env2
	}
	fireRule(r, env)
	return true
}

func checkGuard(g Compound, env Bindings) (env2 Bindings, ok bool) {
	g = Substitute(g, env).(Compound)
	if g.Functor == ":=" || g.Functor == "is" || g.Functor == "=" {
		if !pVar(g.Args[0]) {
			return env, false
		}
		a := Eval(g.Args[1])
		env2 = AddBinding(g.Args[0].(Variable), a, env)
		return env2, true
	}

	t1 := Eval(g)
	switch t1.Type() {
	case BoolType:
		if t1.(Bool) == true {
			return env, true
		}
		return env, false
	case CompoundType:
		biChrList := attributedTerm(t1.(Compound))
		len_chr := len(biChrList)
		if len_chr == 0 {
			return env, false
		}
		for _, chr := range biChrList {
			if Equal(t1, chr) {
				return env, true
			}
		}
		// to do for the operators(@): ==, !=, <, <=, >, >=, =<
		// symmetry: x @ y --> y @ x
		// transitivity: x @ y && y @ z --> x @ z
		//
		// case AtomType, IntType, FloatType, StringType:
		//	case ListType:
		//	case VariableType:
	}
	return env, false
}

func pVar(t Term) bool {
	if t.Type() == VariableType {
		return true
	}
	return false
}

func fireRule(rule *chrRule, env Bindings) {
	return
}
