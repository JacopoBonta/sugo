---
id: ts-codestyle
title: Cubbit Typescript style rules
sidebar_label: Typescript
slug: /guidelines/codestyle/ts
---

## General

For what concerns JS/TS rules the following main rules apply:

## Semicolons are mandatory

Clear code is easy to read and not ambiguous, the absence of semicolons may cause ambiguity and decrease code readability.

✅ Correct

```typescript
const name = 'luke';
const area = width * height;

if(name === 'luke)
    alert('message');
```

❌ Incorrect

```typescript
const name = 'luke'
const area = width * height

if(name === 'luke)
    alert('message')
```

🧐 Example

```typescript
const a = 1
const b = 2
const c = a + b
(a + b).toString()
```

raises a `TypeError: b is not a function`

This rule has a single exception, no `;` after `}``, only in JSON object`};` is mandatory.

✅ Correct

```typescript
const json = {
    cat_name: 'Lucas',
    dog_name: 'Paul',
};
```

❌ Incorrect

```typescript
function like(cat_name: string, dog_name: string): string
{
    console.log(cat_name, 'likes', dog_name);
};
```

## Allman style for braces

Improves the readability of the code, and makes it easier to detect mistakes or errors in the code flow.

✅ Correct

```typescript
while(x == y)
{
    something();
    something_else();
}
```

❌ Incorrect

```typescript
while(x == y){
    something();
    something_else();
}
```

🧐 Example (find the bug)

```typescript
while(x == y)
    something();
    something_else();

    if(x == z){
        z++;
}
```

Read this block, it's hard to guess where the body of the function start

```typescript
public ResultType arbitraryMethodName(
    FirstArgumentType firstArgument,
    SecondArgumentType secondArgument,
    ThirdArgumentType thirdArgument) {
    LocalVariableType localVariable =
        method(firstArgument, secondArgument);
    if(localVariable.isSomething(
        thirdArgument, SOME_COSTANT)) {
        doSomething(localVariable);
    }
    return localVariable.getSomething();
    }
```

```pseudocode
XXXXXX XXXXXXX XXXXXXXXXXXXXXXXXXXXXX
    XXXXXXXXXXXXXXXXXX XXXXXXXXXXXXXX
    XXXXXXXXXXXXXXXXXXXX XXXXXXXXXXXXXX
    XXXXXXXXXXXXXXXXXX XXXXXXXXXXXXXX
    XXXXXXXXXXXXXXX XXXXXXXXXXXX
        XXXXXXXXXXXXXXX XXXXXXXXXXX
    XX XXXXXXXXXXXXXXXX XXXXXXXXXXXX
        XXXXXXXXXX XXXXXXXXXX
        XXXXXXXXX XXXXXXXXX

    XXXXXXX XXXXXXXXX XXXXXXXXXXX
```

```pseudocode
XXXXXX XXXXXXX XXXXXXXXXXXXXXXXXXXXXX
    XXXXXXXXXXXXXXXXXX XXXXXXXXXXXXXX
    XXXXXXXXXXXXXXXXXXXX XXXXXXXXXXXXXX
    XXXXXXXXXXXXXXXXXX XXXXXXXXXXXXXX

    XXXXXXXXXXXXXXX XXXXXXXXXXXX
        XXXXXXXXXXXXXXX XXXXXXXXXXX
    XX XXXXXXXXXXXXXXXX XXXXXXXXXXXX
        XXXXXXXXXX XXXXXXXXXX
        XXXXXXXXX XXXXXXXXX

    XXXXXXX XXXXXXXXX XXXXXXXXXXX

```

Even without code is clear and easy to read the second one.

### Single line instruction

Single-line instruction does not require brackets, (and must be omitted)

✅ Correct

```typescript
if(x == y)
    something();
```

## UpperCamelCase

When we define a class we use the upper camel case syntax, this way the class is easy to recognize across the code and does not confuse.

✅ Correct

```typescript
public class CubbitTransferQueue
{

}
```

❌ Incorrect

```typescript
public class Cubbittransferqueue
{

}

public class Cubbit_transfer_queue
{

}
```

## snake_case for class members

When we define class members, variables functions, and methods names we use the `snake_case` style.

We try to use meaningful variables name, even if it cost longer words, the variable of our system should always be clear easy to debug, and meaningful.

✅ Correct

```typescript
const square_area = side * side;
const number_of_collaborator = 15 * clients;

if(password_hash === password)

collaborators.map((collaborator) => collaborator.subscription = 'erased');
```

❌ Incorrect

```typescript
const squareArea = s * s;
const numberOfCollaborator = 15 * clients;

if(pswd === pswdHash)

collaborators.map((c) => c.subs = 'erased');
```

`subs`? `c`?

those are quite common _mistakes_ we made when we use lambda function, we are brought to believe that due to the fact it's just a lambda this rule should not be used (false), even if the variable is used once on a specific domain, using the full version, easy to read, easy to understand not ambiguous is mandatory in cubbit!

🧐 Example

```typescript
collaborators.map((c) => c.subs = 'erased');
```

## No space before and after braces

The syntax should be compact, avoiding unnecessary complexity.

✅ Correct

```typescript
if(a === b)

while(a)
```

❌ Incorrect

```typescript
if( a === b )

if (a === b)

while (a)
```

## Spaces before and after operators

When some logic is involved we must focus on the code and an easy-to-read approach where each operator can be certainly determined without hesitation to speed up the review and coding.

✅ Correct

```typescript
const area = a + b;

const test = a === b;

if(a >= b)

for(let i = 0; i < n; i++)

collaborators.map(collaborator => collaborator.pay = 1);
collaborators.map(collaborator => {collaborator.pay = 1});
```

❌ Incorrect

```typescript
const area = a+b;

const test = a===b;

const test=a===b;

if(a>=b)

for(int i=0;i<n;i++)

collaborators.map(collaborator=> collaborator.pay = 1);
collaborators.map(collaborator =>collaborator.pay = 1);
collaborators.map(collaborator =>{ collaborator.pay = 1});
```

🧐 Example

```typescript
a=b+c++*2/3
```

## Spaces after comma

✅ Correct

```typescript
Object.invoke(like, cat, dog);
```

❌ Incorrect

```typescript
Object.invoke(like,cat,dog);
Object.invoke(like ,cat ,dog);
Object.invoke(like , cat , dog);
```

## Try catch block

✅ Correct

```typescript
try
{

}
catch(error)
{

}
```

❌ Incorrect

```typescript
try
{

}catch(error)
{

}
```

## Private members and methods start with _

✅ Correct

```typescript
public class Cubbit
{
    private _calculation()
    {
        this.price = 12 * this.other;
    }
}
```

❌ Incorrect

```typescript
public class Cubbit
{
    private calculation()
    {
        this.price = 12 * this.other;
    }
}

public class Cubbit
{
    public _calculation()
    {
        this.price = 12 * this.other;
    }
}
```

## Public members and methods come before private members and methods

The goal is to expose the public interface of a class as the first thing

At the top of the class/file all the public methods so when you start reading the document you easily can use the file. Then for maintenance and further development all the other private methods that the class/files depend on.

✅ Correct

```typescript
public class Cubbit
{
    public price()
    {
        return {price: this._calculation()};
    }

    private _calculation()
    {
        return 12 * this.other;
    }
}
```

❌ Incorrect

```typescript
public class Cubbit
{
    private _calculation()
    {
        return 12 * this.other;
    }

    public price()
    {
        return {price: this._calculation()};
    }
}
```

From time to time, you will be tempted to put the private call as the next function even if after that there will be other public functions, the best solution is to order the private function based on where they get used from the public functions.

## One class per file

The goal here is to enforce smaller files and decouple responsibilities, moreover, this may cause some headaches trying to understand where a class is.

✅ Correct

```typescript
// filename CubbitStatus.ts

class CubbitStatus
{

}
```

❌ Incorrect

```typescript
// filename CubbitStatus.ts

class CubbitStatus
{

}

class Status
{

}
```

`.tsx` files and exceptions, when we speak about components this rule may be confusing at first, in fact, a single component should be self-contained, and sometimes due to the presence of a Styled Component, a single file may be responsible for more than one component.
In this case, a single file should never be the host of multiple `"functional"` components, styled-components do not count.
(obviously export occurs only if `ListOfFile` should be used somewhere else)

🧐 Correct example

```typescript
// FileViewer.tsx

export const ListOfFile = styled.div`
    color: red;
`;

export function FileViewer()
{
    return <div>
        <ListOfFile>This is still a single class file</ListOfFile>
    </div>;
}

```

## Empty lines before and after control blocks

- An empty line before and after control blocks.
- No empty line if the control block is on top of a code block
- No brackets if the control block (if, for, etc..) is just one line of code (the goal here is to get used to writing shorter blocks using negative logic and make the code more readable)

✅ Correct

```typescript
const a = b + 15;

if(a > 20)
    do_things();

b++;

function test(a: string)
{
    if(a === 'casa')
        do_other_things();
}
```

❌ Incorrect

```typescript
const a = b + 15;
if(a > 20)
    do_things();
b++;


const a = b + 15;
if(a > 20)
    do_things();
if(a < 10)
    do_other_things();

function test(a: string)
{

    if(a === 'casa')
        do_other_things();

}
```

## Negative logic

✅ Correct

```typescript
function test(a: string)
{
    if(a === '')
        return;

    do_other_things();
    do_more_stuff();
    another_one();
}
```

❌ Incorrect

```typescript
function test(a: string)
{
    if(a !== '')
    {
        do_other_things();
        do_more_stuff();
        another_one();
    }
}
```

🧐 Example

```typescript
function test_ugly(a: string)
{
    if(a !== '')
    {
        if(a !== 'casa')
        {
            if(a.length > 3)
            {
                do_other_things();
                do_more_stuff();
                another_one();
            }
        }
    }
}


function test_clean(a: string)
{
    if(a === '')
        return;

    if(a === 'casa')
        return;

    if(a.length <= 3)
        return;

    do_other_things();
    do_more_stuff();
    another_one();
}
```

## Enum uppercase

✅ Correct

```typescript
enum Polygon
{
    SQUARE='square',
    TRIANGLE='triangle',
    HEXAGON='hexagon'
}

enum Status
{
    GOOD,
    BAD,
    VERY_BAD
}
```

❌ Incorrect

```typescript
enum Polygon
{
    Square='square',
    Triangle='triangle',
    hexagon='hexagon'
}
```

Remember to give an empty default status for protobuf enum!
🧐 Example

```typescript
enum Status
{
    UNDEFINED_STATUS,
    GOOD,
    BAD,
    VERY_BAD
}
```

## Import without space

✅ Correct

```typescript
import {test} 'test';
```

❌ Incorrect

```typescript
import { test } 'test';
```

## Never use any

When you can insert `any` instead of defining a proper type please `DON'T`.

I mean never use `any`, no matter what or how hard is to properly find a type, this shortcut just blew up in the future so please, spend some time finding a proper type for your call. If `any` is the only possible solution please provide a comment or at least explain why in your PR.
But every time `any` show up in the Cubbit code base we just insert a debt someone else should fix later, so feel free to ask for help from some seniors to find a proper type.

✅ Correct

```typescript
function sum(a: string, b: string): string
function sum(a: number, b: number): number
{
    return a + b;
}
```

❌ Incorrect

```typescript
function sum(a: string, b: number): any
{
    return a + b;
}
```
