#include "libwormhole_william.h"

#include <stdio.h>

int main(void) {
    void *client = (void *)NewClient();

    char *msg = "Hello world";
    char *code = "42-foo-bar";

    int r = ClientSendText(client, msg, &code);
    if (r < 0) {
        fprintf(stderr, "error sending text\n");
    }
    fprintf(stderr, "ClientSendText returned %d\n", r);

    return 0;
}
