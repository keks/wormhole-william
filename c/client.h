# include <stdint.h>

typedef struct client_config {
    char *app_id;
    char *rendezvous_url;
    char *transit_relay_url;
    int32_t passphrase_length;
} client_config;

typedef void (*callback)(void*, int32_t);
void call_callback (callback cb, void *value, int32_t err_code);