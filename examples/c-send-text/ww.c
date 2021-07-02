#include "libwormhole_william.h"

#include <stdio.h>
#include <stdlib.h>

int main(void) {
    void *client = (void *)NewClient();
    int ctxIndex = NewContext();

    char *msg = "Hello world";
    char *code = (char *) malloc(sizeof(char) * 100);

    int r = ClientSendText(client, ctxIndex, msg, &code);
    if (r < 0) {
        fprintf(stderr, "error sending text\n");
    }
    fprintf(stderr, "ClientSendText returned the code %s\n", &code[0]);

    free(code);
    return 0;
}
