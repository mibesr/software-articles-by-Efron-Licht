// in this example, we'll make 'procedures' (that is, functions).
// we are now approaching a 'real' computer program.
// all functions will have a uniform ABI (Application Binary Interface).
//
// # Function ABI
// The variables A[0x0]..A[0xF] are used to pass arguments to functions.
// The variable R[0x0]..R[0xF] are used to return values from functions.
// The variable SCRATCH[0x0]..SCRATCH[0xF] are used as scratch space.
package main

import "os"

// the gotwo virtual machine has a fixed set of global variables.
// these are the registers and memory: for now, don't worry about the distinction, and just reinterpret "register" as "variable".
var (
	A   [16]int  // procedure (function) argument registers. this is often re-used as scratch space. (in the 'real' world, arguments and return values share one set of registers, but we're not going to do that here.)
	R   [16]int  // procedure (function) return values registers.
	D   int      // return address stack depth. if < 0, the program exits.
	RET [255]int // return address stack memory. the return address of our current caller is at RET[D].
	io  [16]byte // memory, used as a buffer for I/O.

)

// RETURN LABELS, used as psuedo-program-counter values.
// these are the 'public' procedures that can be called from anywhere.
// each L_XXX corresponds to a procedure XXX: e.g. L_FIB corresponds to the procedure FIB.
// See the body of RETURN for more information.
//
// # Naming
//
// we use the following naming conventions:
//   - L_XXX is the RETURN LABEL corresponding to the procedure XXX.
//   - XXX is the procedure itself.
//
// That is, the label L_EXIT corresponds to the procedure EXIT.
// If a procedure contains a loop, we append _LOOP to the label name.
// If that loop is nested, we add _0, _1, _2, etc. to the loop name:
//
//   - L_POWER_LOOP is the RETURN LABEL corresponding to the loop in the POWER procedure.
//   - POWER_LOOP is the loop in the POWER procedure.
//
// Not all LABELS have a corresponding constant: only those that are "public" procedures.

// "system call" procedures that have special behavior (i.e, call actual go functions, not gotwo 'procedures')
// these are written as close to "go two" style as possible, but cheat at the edges in order to provide input and output.
// system calls count DOWN from zero
const (
	_       = -iota // skip the first value: it's always zero.
	L_EXIT          // exit the program with code specified in A[0].
	L_PRINT         // print R[0]..R[0xF] to the screen until a zero is encountered. uses a "system call" to write to stdout.
	L_SCAN          // read up to 16 bytes from stdin and store them in R[0]..R[0xF]... a zero byte terminates the input. uses a "system call" to read from stdin.
)

// "public" procedures that can be called from anywhere.
const (
	L_ENTRY       = iota // skip L_ENTRY: it's the default value of RET[0].
	L_MUL                // mul(A[0], A[1]) -> R[0].
	L_FIB                // fib(A[0]) -> R[0].
	L_POWER              // power(A[0], A[1]) -> R[0].
	L_DIVMOD             // div(A[0], A[1]. A[0] / A[1]) -> R[0], A[0] % A[1] -> R[1].
	L_READPAGE           // read a page from disk.
	L_POWER_LOOP         // loop in the POWER procedure, used as a "return label" for MUL, among others.
	L_ATOI               // convert a string of up to 16 decimal digits (in A[0]..A[0xF]) to an integer. the result, if any, is in R[0]. if the string is invalid, R[0] is zero and R[1] is nonzero.
	L_ATOI_RETURN        // return label for ATOI, used by MUL.

)

// style:
// in order to make the body of our procedures easier to follow,
// we enclose each procedure in a 'bare' block:
//
//	 PROCEDURE_NAME:
//		{
//		   if some_condition {
//				goto RETURN
//		   }
//		   goto ANOTHER_LABEL
//		}
//
// Since we have only global variables, this is unnecessary, but it makes the code easier to read.
func main() {
	// start of the program.
	RET[0] = L_ENTRY

RETURN:
	/* ----------------- RETURN ----------------- */
	// all functions return here.
	// at this point:
	// - D is the depth of the return address stack.
	// - RET[D] contains the RETURN LABEL for the next jump.
	// - R[0]..R[0xF] contain the return values of the called procedure, if any.
	// - A[0]..A[0xF] may contain any data: they are not guaranteed to hold the originally passed arguments.

	/* design note:
	    // this is a "trampoline".
	   	// an 'ordinary' virtual machine would let us jump to a specific address.
	   	// however, go doesn't have computed gotos: we must jump to EXACTLY one label.
	   	// we could have each function contain a jump table for every possible caller, but this is not extensible and would hugely bloat the code.
	   	// instead, we have exactly one place, RETURN, where all functions return to.
	   	// they then look at RET[D] to see where they should go next, and 'bounce' to that label: you only jump
	   	// on to RETURN in order to bounce somewhere else, hence the term 'trampoline'.
	*/

	D-- // decrement the return address stack depth.
	switch RET[D+1] {
	case L_ATOI: // convert a string of up to 16 hex digits to an integer. the input is in A[0]..A[0xF]. the result, if any, is in R[0]. if the string is invalid, R[0] is zero and R[1] is nonzero.
		goto ATOI
	case L_ENTRY: // entrypoint of the program.
	// TODO: add initialization conditions here.
	case L_EXIT: // end of the program. return the value in R[0].
		goto EXIT
	case L_FIB: // generalized fibonacci function: A[0]: n, A[1]: "current" value, A[2]: "previous" value. For ordinary fib(n), A[0]=n, set A[1] = 1, A[2] = 0.
		goto FIB
	case L_MUL: // multiply(A[0], A[1]) -> R[0].
		goto MUL
	case L_POWER_LOOP: // part of the power function: used as a "return label" for MUL, among others.
		goto POWER_LOOP
	case L_POWER: // A[0] ^ A[1] -> R[0].
		goto POWER
	case L_DIVMOD: // division and modulus, stored in R[0] and R[1] respectively.
		goto DIV
	case L_PRINT: // print A[0]..A[0xF] to the screen until a zero is encountered or 16 bytes are written. return the number of bytes written in R[0].
		goto PRINT
	case L_SCAN: // read up to 16 bytes from stdin and store them in R[0]..R[0xF]... a zero byte terminates the input.
		goto SCAN
	case L_ATOI_RETURN:
		goto ATOI_AFTER_MUL

	}

	// --------- PROCEDURES ---------
	// each procedure is a block of code that can be called from anywhere.
	// a procedure operates on up to 16 arguments (stored in global registers A[0]..A[0xF]) and returns up  to 16 values (stored in global registers R[0]..R[0xF]).
	// procedures should not touch RET or D except to call other procedures.

ATOI: // convert a string of up to 16 decimal digits ('0', '1', '2'... '9') to an integer. the string may be terminated by a zero byte.
	// the input is in A[0]..A[0xF]. the result, if any, is in R[0]. if the string is invalid, R[0] is zero and R[1] is nonzero, and R[2] is the index where the error occurred (0..15).
	// this doesn't handle hexadecimal digits, negative numbers, or whitespace.
	const ( // error codes.
		atoiErrNone    = 0
		atoiErrEmpty   = 1
		atoiErrInvalid = 2
	)
	// register aliases for returns
	const (
		n     = 0 // used by MUL: be careful of clobbering.
		err   = 1 // used by MUL: be careful of clobbering.
		i     = 2 // loop counter. not used by MUL.
		total = 3 // not used by MUL.
		digit = 4 // current digit (not the index, the actual digit). not used by MUL.
	)

	R[n] = 0
	R[err] = atoiErrEmpty
	if A[0] == 0 {
		goto RETURN // empty string.
	}
	R[total] = 0
	// fallthrough to ATOI_LOOP.
ATOI_LOOP:
	{
		A[digit] = A[i] // save the current digit.
		// check validity.
		R[err] = atoiErrInvalid
		// bounds checks
		if A[0] < '0' {
			goto RETURN
		}
		if A[0] > '9' {
			goto RETURN
		}
		// convert to integer.
		R[digit] = int(A[0] - '0')
		R[err] = atoiErrNone

		// multiply the total by 10, then add the digit.
		// set up the arguments for the MUL subroutine.
		A[0] = R[total]
		A[1] = 10
		// push the return address and arguments onto the stack.
		D++                    // one deeper
		RET[D] = L_ATOI_RETURN // MUL should return to ATOI_RETURN.
		goto MUL               // call the subroutine. when it returns, RET[D+1] will be L_ATOI_RETURN, continuing the loop.
	}
ATOI_AFTER_MUL:
	{
		const (
			err   = 1  // register aliases for returns.
			i     = 8  // register alias for loop counter: we know these aren't used by MUL.
			total = 9  // register alias for total: we know these aren't used by MUL.
			digit = 10 // register alias for current digit: we know these aren't used by MUL.
		)
		R[total] = R[total]*10 + R[digit]
		R[i]++
		if R[i] == 16 { // we've read 16 digits: stop.
			goto RETURN
		}
		if A[R[i]] == 0 { // we've reached the end of the string. stop.
			goto RETURN
		}
		goto ATOI_LOOP
	}

MUL: // multiply(A[0], A[1]) -> R[0].
	{
		const n, m = 0, 1
		R[0] = 0 // clear the return value so we can begin accumulating.
	MUL_LOOP:
		if A[m] == 0 {
			goto RETURN
		}
		R[0] += A[n]
		A[m]--
		goto MUL_LOOP
	}

DIV: // left as an exercise for the reader.
	{
		// TODO
		goto RETURN
	}

POWER:
	{
		// power(n, m).
		// calculate n^m, placing the result in R[0].
		// we repeatedly multiply n by itself m times.
		const cur, base, exp = 15, 14, 13 // we know these aren't used by MUL.
		R[0] = 1                          // clear the return value.
		A[base] = A[0]
		A[exp] = A[1]
	} // fallthrough to POWER_LOOP.
POWER_LOOP: // loop over the exponent, multiplying the base by itself.
	{
		const cur, base, exp = 15, 14, 13 // we know these aren't used by MUL, so they're O.K to reuse.
		A[cur] = R[0]                     // save the current result.
		if A[exp] == 0 {
			goto RETURN
		}
		A[exp]-- // decrement the exponent so the loop terminates.

		// set up the arguments for the MUL subroutine.
		A[0] = A[cur]
		A[1] = A[base]
		// push the return address and arguments onto the stack.
		D++                   // one deeper
		RET[D] = L_POWER_LOOP // MUL should return to POWER_LOOP.
		goto MUL              // call the subroutine. when it returns, RET[D+1] will be L_POWER_LOOP, continuing the loop.
	}

FIB: // generalized fibonacci function: A[0]: n, A[1]: "current" value, A[2]: "previous" value. For ordinary fib(n), A[0]=n, set A[1] = 1, A[2] = 0.
	{
		const n, cur, prev, tmp = 0, 1, 2, 3
	FIB_LOOP:
		if A[n] == 0 {
			R[0] = A[cur]
			goto RETURN
		}
		A[tmp] = A[cur] // no multiple assignment, so use a temporary variable as scratch space.
		A[cur] = A[prev] + A[cur]
		A[prev] = A[tmp]
		A[n]--
		goto FIB_LOOP
	}

PRINT: // print up to sixteen characters to the screen, specified by A[0]..A[0xF]. a zero byte terminates the string, and the number of bytes written is returned in R[0] (but you should already know that from the calling convention).
	{
		io[0], io[1], io[2], io[3] = 0, 0, 0, 0
		io[4], io[5], io[6], io[7] = 0, 0, 0, 0
		io[8], io[9], io[10], io[11] = 0, 0, 0, 0
		io[12], io[13], io[14], io[15] = 0, 0, 0, 0
		/*
			design note:
			all of A[0]..A[0xF] might already be used. we can't use them as scratch space, but we CAN use R[0]..R[0xF], since we're not returning anything.
			we let R[0] be our loop counter. conveniently, this also means we "return" the number of bytes written. neat, huh?
		*/
		const i = 0 // loop counter.

	PRINTCHAR:
		if A[0] == 0 {
			goto RETURN
		}
		// convert to 7-bit ASCII.
		const ASCII = 0b0111_1111
		A[i] &= ASCII
		io[R[i]] = byte(A[i])
		R[i]++

		if R[i] < 15 {
			goto PRINTCHAR
		}
		// actually write the bytes via "system call."
		_, _ = os.Stdout.Write(io[:A[i]]) // cheating: we allow os.Stdout.Write as a "system call".
		goto RETURN
	}

SCAN: // read up to 16 bytes from stdin and store them in R[0]..R[0xF]... a zero byte terminates the input.
	{
		// clear memory.
		io[0], io[1], io[2], io[3] = 0, 0, 0, 0
		io[4], io[5], io[6], io[7] = 0, 0, 0, 0
		io[8], io[9], io[10], io[11] = 0, 0, 0, 0
		io[12], io[13], io[14], io[15] = 0, 0, 0, 0

		// "system call" to read from stdin.
		_, _ = os.Stdin.Read(io[:])

		// set return values.
		R[0], R[1], R[2], R[3] = int(io[0]), int(io[1]), int(io[2]), int(io[3])
		R[4], R[5], R[6], R[7] = int(io[4]), int(io[5]), int(io[6]), int(io[7])
		R[8], R[9], R[10], R[11] = int(io[8]), int(io[9]), int(io[10]), int(io[11])
		R[12], R[13], R[14], R[15] = int(io[12]), int(io[13]), int(io[14]), int(io[15])
		goto RETURN
	}

EXIT: // exit the program with code specified in A[0].
	{
		os.Exit(A[0]) // "system call" to exit.
	}
}
