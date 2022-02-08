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

void free_result(result_t *result) {
  /*debugf("Freeing result located at %p", result);*/
  if (result != NULL) {
    if (result->err_string != NULL) {
      free(result->err_string);
    }

    if (result->file != NULL) {
      if (result->file->data != NULL) {
        free(result->file->data);
      }

      if (result->file->file_name != NULL) {
        free(result->file->file_name);
      }

      free(result->file);
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
