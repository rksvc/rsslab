#include "sqlite3ext.h"
#include <ctype.h>
#include <stddef.h>
SQLITE_EXTENSION_INIT1

static fts5_api *fts5_api_from_db(sqlite3 *db) {
    fts5_api *pRet = NULL;
    sqlite3_stmt *pStmt = NULL;

    if (sqlite3_prepare(db, "SELECT fts5(?1)", -1, &pStmt, 0) == SQLITE_OK) {
        sqlite3_bind_pointer(pStmt, 1, (void *)&pRet, "fts5_api_ptr", NULL);
        sqlite3_step(pStmt);
    }
    sqlite3_finalize(pStmt);
    return pRet;
}

typedef struct {
    int (*xTokenize)(void *pCtx, int flags, const char *pText, int nText,
                     int (*xToken)(void *pCtx, int tflags, const char *pToken,
                                   int nToken, int iStart, int iEnd));
} simple_tokenizer;

typedef enum { ALPHA, DIGIT, SPACE, OTHER } category;

static category category_of(char c) {
    return isalpha(c)                 ? ALPHA
           : isdigit(c)               ? DIGIT
           : isspace(c) || iscntrl(c) ? SPACE
                                      : OTHER;
}

static int len(unsigned char b) {
    return b >= 0xf0 ? 4 : b >= 0xe0 ? 3 : b >= 0xc0 ? 2 : 1;
}

static int simple_tokenize(void *pCtx, int flags, const char *pText, int nText,
                           int (*xToken)(void *pCtx, int tflags,
                                         const char *pToken, int nToken,
                                         int iStart, int iEnd)) {
    int iStart = 0;
    int iEnd = 0;
    category category;

    while (iEnd < nText) {
        category = category_of(pText[iStart]);
        if (category == OTHER)
            iEnd += len(pText[iStart]);
        else
            while (++iEnd < nText && category_of(pText[iEnd]) == category)
                ;
        if (category != SPACE) {
            int rc;
            if (category == ALPHA) {
                char *p =
                    (char *)sqlite3_malloc((iEnd - iStart) * sizeof(char));
                if (!p)
                    return SQLITE_ERROR;
                for (int i = 0; i < iEnd - iStart; ++i)
                    p[i] = tolower(pText[iStart + i]);
                rc = xToken(pCtx, 0, p, iEnd - iStart, iStart, iEnd);
                sqlite3_free(p);
            } else {
                rc = xToken(pCtx, 0, pText + iStart, iEnd - iStart, iStart,
                            iEnd);
            }
            if (rc != SQLITE_OK)
                return rc;
        }
        iStart = iEnd;
    }

    return SQLITE_OK;
}

static int fts5_simple_create(void *pCtx, const char **azArg, int nArg,
                              Fts5Tokenizer **ppOut) {
    simple_tokenizer *p =
        (simple_tokenizer *)sqlite3_malloc(sizeof(simple_tokenizer));
    if (!p)
        return SQLITE_ERROR;
    p->xTokenize = simple_tokenize;
    *ppOut = (Fts5Tokenizer *)(p);
    return SQLITE_OK;
}

static void fts5_simple_delete(Fts5Tokenizer *p) { sqlite3_free(p); }

static int fts5_simple_tokenize(Fts5Tokenizer *pTokenizer, void *pCtx,
                                int flags, const char *pText, int nText,
                                int (*xToken)(void *, int, const char *, int,
                                              int, int)) {
    simple_tokenizer *p = (simple_tokenizer *)pTokenizer;
    return p->xTokenize(pCtx, flags, pText, nText, xToken);
}

#ifdef _WIN32
__declspec(dllexport)
#endif
    int sqlite3_simpletokenizer_init(sqlite3 *db, char **pzErrMsg,
                                     const sqlite3_api_routines *pApi) {
    SQLITE_EXTENSION_INIT2(pApi);

    fts5_tokenizer tokenizer = {fts5_simple_create, fts5_simple_delete,
                                fts5_simple_tokenize};
    fts5_api *api = fts5_api_from_db(db);
    return api ? api->xCreateTokenizer(api, "simple", (void *)api, &tokenizer,
                                       NULL)
               : SQLITE_OK;
}
