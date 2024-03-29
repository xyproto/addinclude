Addinclude
==========

Utility that can add `#include`s to `.c`, `.cpp` or `.h` files, with relatively smart placement.

Example use
-----------

    addinclude my.c stdin
    addinclude my.cpp memory

Smart placement
---------------

    addinclude my.h stdin

Changes my.h from:

    #ifdef blabla
    #endif

To:

    #ifdef blabla
    #include <stdin.h>
    #endif

You can place includes at the very top of the file with `-t`. There are several other options.

C++ headers
-----------

* Use the `-c++` flag for not expanding include names when adding them to files not ending with `.cpp`.
* For example, `memory` will not be expanded to `memory.h`.

General info
------------

* Version: 1.2.0
* License: GPL2
