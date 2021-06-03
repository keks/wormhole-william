#include "client.h"

void call_callback(callback cb, void *result, int32_t err_code) {
    cb(result, err_code);
};