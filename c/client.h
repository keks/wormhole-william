#include <stdint.h>

#ifndef CLIENT_INCLUDED
#define CLIENT_INCLUDED
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
  void *context;
} file_metadata_t;

typedef struct {
  int32_t err_code;
  char *err_string;
  char *received_text;
} result_t;

typedef struct {
  int32_t error_code;
  char *error_string;
  char *code;
} codegen_result_t;

typedef struct {
  int64_t transferred_bytes;
  int64_t total_bytes;
} progress_t;

typedef void (*async_cb)(void *context, result_t *result);
typedef void (*progress_cb)(void *context, progress_t *progress);
typedef int (*readf)(void *context, uint8_t *buffer, int length);
typedef int64_t (*seekf)(void *context, int64_t offset, int whence);
typedef void (*writef)(void *context, uint8_t *buffer, int length);
typedef void (*file_metadata_cb)(void *context, file_metadata_t *metadata);

void call_callback(void *context, async_cb cb, result_t *result);
void update_progress(void *context, progress_cb cb, progress_t *progress);
int call_read(void *context, readf f, uint8_t *buffer, int length);
int64_t call_seek(void *context, seekf f, int64_t offset, int whence);
void update_metadata(void *context, file_metadata_cb cb,
                     file_metadata_t *metadata);
void call_write(void *context, writef f, uint8_t *buffer, int length);

void free_result(result_t *result);
void free_codegen_result(codegen_result_t *codegen_result);
#endif
