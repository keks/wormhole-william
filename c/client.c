#include "client.h"

#include <stdlib.h>

void call_notify_result(client_context_t ctx, notify_resultf f,
                        result_t *result) {
  f(ctx, result);
}

void call_update_progress(client_context_t ctx, update_progressf pcb,
                          progress_t *progress) {
  pcb(ctx, progress);
}

void call_update_metadata(client_context_t ctx, update_metadataf mdf,
                          file_metadata_t *metadata) {
  return mdf(ctx, metadata);
}

int call_read(client_context_t ctx, readf f, uint8_t *buffer, int length) {
  return f(ctx, buffer, length);
}

int64_t call_seek(client_context_t ctx, seekf f, int64_t offset, int whence) {
  return f(ctx, offset, whence);
}

int call_write(client_context_t ctx, writef f, uint8_t *buffer, int length) {
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
    if (result->result_type == CodeGenSuccessful) {
      free(result->generated.code);
    } else {
      free(result->error.error_string);
    }
    free(result);
  }
}
