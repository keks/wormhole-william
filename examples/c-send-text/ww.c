#include "libwormhole_william.h"

#include <stdio.h>
#include <stdlib.h>

int main(void) {
    void *client = (void *)NewClient();
    int ctxIndex = NewContext();

    char *msg = "Hello world";
    char *code = (char *) malloc(sizeof(char) * 100);
    char *side = ClientSideID();
    char *appid = ClientAppID(client);

    int rcIndex = ClientGetCode(client, ctxIndex, side, appid, &code);

    if (rcIndex == -1) {
        fprintf(stderr, "error sending text\n");
    }
    fprintf(stderr, "ClientGetCode returned the code %s\n", &code[0]);

    int status = ClientSendTextMsg(client, ctxIndex, rcIndex, side, appid, code, msg);
    if (status < 0) {
        fprintf(stderr, "error sending text message\n");
    }
    DeleteContext(ctxIndex);

    free(code);
    return 0;
}
