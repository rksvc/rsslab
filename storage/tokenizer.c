#include "sqlite3ext.h"
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

typedef struct siyuan_tokenizer {
    int (*xTokenize)(void *pCtx, int flags, const char *pText, int nText,
                     int (*xToken)(void *pCtx, int tflags, const char *pToken,
                                   int nToken, int iStart, int iEnd));
} siyuan_tokenizer;

static const char CHARACTER_BYTES_FOR_UTF8[256] = {
    1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
    1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
    1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
    1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
    1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
    1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
    2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
    4, 4, 4, 4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0};

static int siyuan_tokenize(void *pCtx, int flags, const char *pText, int nText,
                           int (*xToken)(void *pCtx, int tflags,
                                         const char *pToken, int nToken,
                                         int iStart, int iEnd)) {
    int iStart = 0;
    int iEnd = 0;

    while (iEnd < nText) {
        int length = CHARACTER_BYTES_FOR_UTF8[(
            unsigned int)(unsigned char)pText[iStart]];
        iEnd += length;
        if (length == 0 || iEnd > nText)
            return SQLITE_ERROR;

        int rc = xToken(pCtx, 0, pText + iStart, length, iStart, iEnd);
        if (rc != SQLITE_OK)
            return rc;
        iStart = iEnd;
    }

    return SQLITE_OK;
}

static int fts5_siyuan_create(void *pCtx, const char **azArg, int nArg,
                              Fts5Tokenizer **ppOut) {
    siyuan_tokenizer *p =
        (siyuan_tokenizer *)sqlite3_malloc(sizeof(siyuan_tokenizer));
    if (!p)
        return SQLITE_ERROR;
    p->xTokenize = siyuan_tokenize;
    *ppOut = (Fts5Tokenizer *)(p);
    return SQLITE_OK;
}

static void fts5_siyuan_delete(Fts5Tokenizer *p) { sqlite3_free(p); }

static int fts5_siyuan_tokenize(Fts5Tokenizer *pTokenizer, void *pCtx,
                                int flags, const char *pText, int nText,
                                int (*xToken)(void *, int, const char *, int,
                                              int, int)) {
    siyuan_tokenizer *p = (siyuan_tokenizer *)pTokenizer;
    return p->xTokenize(pCtx, flags, pText, nText, xToken);
}

#ifdef _WIN32
__declspec(dllexport)
#endif
    int sqlite3_siyuantokenizer_init(sqlite3 *db, char **pzErrMsg,
                                     const sqlite3_api_routines *pApi) {
    SQLITE_EXTENSION_INIT2(pApi);

    fts5_tokenizer tokenizer = {fts5_siyuan_create, fts5_siyuan_delete,
                                fts5_siyuan_tokenize};
    fts5_api *api = fts5_api_from_db(db);
    return api ? api->xCreateTokenizer(api, "siyuan", (void *)api, &tokenizer,
                                       NULL)
               : SQLITE_OK;
}
