#include "libwormhole_william.h"

#include <stdio.h>

int main(void) {
    void *client = (void *)NewClient();

    char *msg = "Hello world";
    char code[100];

    int r = ClientSendText(client, msg, &code[0]);
    if (r < 0) {
        fprintf(stderr, "error sending text\n");
    }
    fprintf(stderr, "ClientSendText returned the code %s\n", &code[0]);

    return 0;
}
