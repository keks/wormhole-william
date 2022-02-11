#include "client.h"

#include <stdlib.h>

void call_callback(void *ptr, async_cb cb, result_t *result) {
  cb(ptr, result);
}

void update_progress(void *ptr, progress_cb pcb, progress_t *progress) {
  pcb(ptr, progress);
}

int call_read(void *ctx, readf f, uint8_t *buffer, int length) {
  return f(ctx, buffer, length);
}

int64_t call_seek(void *ctx, seekf f, int64_t offset, int whence) {
  return f(ctx, offset, whence);
}

void update_metadata(void *context, file_metadata_cb fmd_cb,
                     file_metadata_t *metadata) {
  return fmd_cb(context, metadata);
}

void call_write(void *ctx, writef f, uint8_t *buffer, int length) {
  return f(ctx, buffer, length);
}

void free_file_metadata(file_metadata_t *fmd) {
  if (fmd != NULL) {
    if (fmd->file_name != NULL) {
      free(fmd->file_name);
    }
    free(fmd);
  }
}

void free_result(result_t *result) {
  if (result != NULL) {
    if (result->err_string != NULL) {
      free(result->err_string);
    }

    if (result->received_text != NULL) {
      free(result->received_text);
    }

    free(result);
  }
}

void free_codegen_result(codegen_result_t *result) {
  if (result != NULL) {
    if (result->code != NULL) {
      free(result->code);
    }
    if (result->error_string != NULL) {
      free(result->error_string);
    }
    free(result);
  }
}
