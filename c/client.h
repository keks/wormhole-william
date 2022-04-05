#include <stdint.h>

#ifndef CLIENT_INCLUDED
#define CLIENT_INCLUDED

#if defined(_WIN32) || defined(_WIN64)
#define DLL_EXPORT __attribute__((dllexport))
#else
#define DLL_EXPORT
#endif

typedef void *client_context_t;

typedef struct {
  char *app_id;
  char *rendezvous_url;
  char *transit_relay_url;
  int32_t passphrase_length;
} client_config;

typedef struct {
  int64_t length;
  char *file_name;
  int32_t download_id;
  client_context_t context;
} file_metadata_t;

typedef enum {
  Success = 0,
  SendFileError = 1,
  ReceiveFileError = 2,
  SendTextError = 3,
  ReceiveTextError = 4,
  TransferRejected = 5,
  TransferCancelled = 6,
} result_type_t;

typedef struct {
  result_type_t result_type;
  char *err_string;
  char *received_text;
} result_t;

typedef enum {
  CodeGenSuccessful = 0,
  FailedToGetClient = 1,
  CodeGenerationFailed = 2
} codegen_result_type_t;

typedef struct {
  codegen_result_type_t result_type;
  client_context_t context;
  struct {
    char *error_string;
  } error;
  struct {
    char *code;
    int32_t transfer_id;
  } generated;
} codegen_result_t;

typedef struct {
  int64_t transferred_bytes;
  int64_t total_bytes;
} progress_t;

typedef void (*notify_resultf)(client_context_t context, result_t *result);
typedef void (*update_progressf)(client_context_t context,
                                 progress_t *progress);
typedef void (*update_metadataf)(client_context_t context,
                                 file_metadata_t *metadata);

typedef int (*readf)(client_context_t context, uint8_t *buffer, int length);
typedef int64_t (*seekf)(client_context_t context, int64_t offset, int whence);
typedef int (*writef)(client_context_t context, uint8_t *buffer, int length);

void call_notify_result(client_context_t context, notify_resultf f,
                        result_t *result);
void call_update_progress(client_context_t context, update_progressf cb,
                          progress_t *progress);
void call_update_metadata(client_context_t context, update_metadataf cb,
                          file_metadata_t *metadata);

int call_read(client_context_t context, readf f, uint8_t *buffer, int length);
int64_t call_seek(client_context_t context, seekf f, int64_t offset,
                  int whence);
int call_write(client_context_t context, writef f, uint8_t *buffer, int length);

DLL_EXPORT void free_result(result_t *result);
DLL_EXPORT void free_codegen_result(codegen_result_t *codegen_result);
#endif
