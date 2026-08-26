---
date: 2024-03-07
---

This ADR registers design decisions regarding the MsgPolicyCmd transaction.
MsgPolicyCmd enables the ACP module to accept Commands to a Policy from users that do not have a Vera account.
This feature works by introducing a payload which contains the command, the actor's DID, metadata and a signature issued by the Actor.
To execute the command the payload's signature is validated using the DID's `VerificationMethod` and a series of checks are performed, iff the payload is valid the ACP module extracts the command and executes it.


Intro
=====

This ADR registers design decisions regarding the MsgPolicyCmd transaction.
MsgPolicyCmd enables the ACP module to accept Commands to a Policy from users that do not have a Vera account.
This feature works by introducing a payload which contains the command, the actor's DID, metadata and a signature issued by the Actor.
To execute the command the payload's signature is validated using the DID's `VerificationMethod` and a series of checks are performed, iff the payload is valid the ACP module extracts the command and executes it.



Body
====

## Removing the need for Vera accounts

Prior to this message, any operation done under a Policy required the Actor to issue a Tx signed from a Vera account.
The ACP Module messages would issue each Vera address a DID which would be used as the Subject for the Relationships.
This system imposed a great limitation upon the identifiers a Policy was able to accept, since effectively only Vera accounts could be Actors under a Policy.

The MsgPolicyCmd solves this problem by decoupling the Tx signer from the Msg issuer by introducing a signed payload to the system.
This signed payload was designed to accept any DID Actor, so long as the DID is resolvable to a DID Document which contains some VerificationMethod / public key.

A related question is why dids as opposed to an arbitrary actor registration system?
DIDs were simple and convenient in that they're a self contained identifier which can be easily resolved to key material for authentication.
An alternative to DIDs would be self signed certificates, however they're often quite bulky in comparasion and often times the additional data is not needed for an application.
As such, this system was designed to support DIDs as first class citzens.

Note that the restriction to DIDs now does not prohibits the use of arbitrary string IDs in the future.
It would be possible to add a mapping layer which could take an arbitrary string and associate it to a DID.

## Signed Payload

Ths signed payload is simply a data-signature pair which contains a Command for the ACP module to execute.
The signature is used to guarantee integrity and authenticate the issuer at the same time.
The nature of DIDs means they are resolved to a Public Key which is used to verify the signature - thus verifying integrity and authenticating all at once.

The signed payload is by design a single use entity which is sent only once, from an Actor to Vera.
Upon accepting a payload, the ACP module must not process it again, otherwise the system is vulnerable to replay attacks.
See section on preventing attacks for an overview of the protection systems in place.

### Payload Fields

- id: id MUST be an UUID v4 identifier to uniquely identify the payload. v4 is chosen to preserve user privacy
- actor: actor MUST be a valid did string which contains a "VerificationMethod".
- issued_height: issued_height acts as metadata for users indicating the Vera block height for which the payload was generated
- expiration_height: expiration_height defines an upperbound at which the payload will be accepted. If Vera's current height is greater than expiration_height, the payload MUST be rejected.
- command: command is an oneof field containing the command the ACP module will execute.

A related question might be: why not JWT fields?
There are two lines of thought to this: a desire (perhaps unjustified) to not tie the payload to JWS only and certain semantics wouldn't translate 1-1, most notably iat, nbf, exp which specifies timestamps (NumericDate) as opposed to block heights.

## Signed Payload Format and Validation

The Signed Payload was designed to support multiple formats.

## JWS

- ignore any field other than alg in header
- only accept web encoded jws

### Security Notes

TLDR: Ignore ALL JOSE headers while verifying a JWS in the ACP module. Verification MUST use the DID specificied in the `actor` field of the payload.

The JOSE standard is infamous for being easily exploitable and difficult to make secure.
Over the years some clever attacks have been executed, such as changing the header alg from a signature algorithm to a MAC.
The issue always boiled down to trusting the JWS when it shouldn't be trusted.
The payload and header are somewhat malleable, despite the signature.
The current best practice is to restrict the JWS to only be validated for a set of previously configured algorithms.

To further complicate matters, JWS JSON Serialization supports a "JWS Unprotected Header" option which opens yet another can of worms.
With all the challenges and nuances, correctly verifying a JWS can be tricky.

In the case of the ACP module there's one additional detail which must be kept in mind.
The Adversary model for ACP is different than the usual Adversary for JWSs out in the wild.

The common use case for JWS is to share a signed and verified payload between applications.
The payload is generated by some trusted party and the verification address the question of "was this payload generated with a key I trust?".
As such, most attacks such as kid trickery or changing from Sign to MAC attempts to trick the code into saying "yes, this JWS was issued with this key".
The assumption is that if the JWS was verified it meant that it was issue by a trusted party and therefore the payload can be trusted.

On the other hand, the ACP module does not care who issued the JWS.
There is no previously known set of trusted parties that could've issued that JWS, they are all self issued / signed.
The JWS serves as an authentication mechanism, proving that the issuer controls the private key matching the public key in the JWS.
An ACP Adversary is playing a completely different game from regulary Adversaries, there is no challenge in making ACP accept a JWS token, any adversary can issue a valid JWS.
The game the Adversary is playing is that of impersonation, an Adversary would succeed if it is able to generate a verifiable JWS (easy) which tricks the ACP module into thinking it was generated by someone else (must be hard), which is quite different from the regular case.
In the regular case the Adversary doesn't care about who the Server thinks the issuer is, it just has to accept the JWS; the ACP adversary can easily make the server accept a JWS, but their challenge is trickying the server into thinking the JWS was issued by another Actor.

Since the JWS header can be used to inject verification information (eg embedding a JWK, kid, x509 urls) it MUST be ignored while validating an ACP JWS, otherwise libraries or incorrect code might use the header data to verify the JWS as though it was issued by the `actor` in the payload.
The JOSE Header can be exploited by the Adversary so they can win the game.



## Replay Attack Protection

- expiration_height
- cache id until expiration_height


## Verification Steps
- extract did from payload
- resolve did 
- extract did's verification method
- verify signature using did's verification method
- verify expeiration_height is less than current height
- verify that id is not in cache of payloads
- accept


Conclusion
==========