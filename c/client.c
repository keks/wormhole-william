#include "client.h"

void call_callback(void *ctx, callback cb, void *result, int32_t err_code) {
  cb(ctx, result, err_code);
}
