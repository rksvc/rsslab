package storage

// #include <sqlite3.h>
//
// extern int sqlite3_siyuantokenizer_init(sqlite3 *, char **, const sqlite3_api_routines *);
//
// void __attribute__((constructor)) init(void) {
//     sqlite3_auto_extension((void *)sqlite3_siyuantokenizer_init);
// }
import "C"
