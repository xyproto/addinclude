Addinclude
==========

A utility to add #includes to .c and/or .h files.

Easy syntax
-----------

    addinclude my.c stdin

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

You can place includes at the top of the file with -t.
There are several other options.

TODO
----
* C++ support

