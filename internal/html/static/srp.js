import { SRPClient } from '@windwalker-io/srp'
import { ArgonWorker, variant } from './argon2ian.async.min'
import { bufToBigint, hexToBuf, hexToBigint } from 'bigint-conversion'


const client = new SRPClient(
  BigInt('0xFFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6BF12FFA06D98A0864D87602733EC86A64521F2B18177B200CBBE117577A615D6C770988C0BAD946E208E24FA074E5AB3143DB5BFCE0FD108E4B82D120A93AD2CAFFFFFFFFFFFFFFFF'),
  5n,
  3673775127127765392429883813496016918310284849659458897286420918568418152734n
)

/** 
 * @param salt {Uint8Array} - cryptographic salt
 * @param identity {string} - Identifier for SRP protocol
 * @param password {string} - Password 
 * @return {Promise<BigInt>}
 * */
client.generatePasswordHash = async function(salt, identity, password) {
  const pw = new TextEncoder().encode(`${identity}:${password}`)
  const worker = new ArgonWorker();
  await worker.ready

  const result = await worker.hash(pw, salt, {
    variant: variant.Argon2id,
    length: 32,
    m: 64 * 1024, // memory cost
    t: 1, // iterations
    p: 4 // threads
  })

  worker.terminate()
  return bufToBigint(result)
}

/** 
 * registerUser will generate a verifier and salt value for a user and POSTs it to the auth api
 * returns true on success, otherwise throws an error containing the error returned by register endpoint
 * @param {string} identifier
 * @param {string} password
 * @returns {boolean}
 * @throws {Error}
 * */
const registerUser = async function(identifier, password) {
  let { salt, verifier } = await client.register(identifier, password)
  const response = await fetch("/api/auth/register", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      "identifier": identifier,
      "salt": salt.toString(16).padStart(32, '0'),
      "verifier": verifier.toString(16),
    }),
  })

  const data = await response.json()
  if (response.status !== 200) {
    throw new Error(data.error)
  }

  return true
}

/** 
 * loginUser handles the login handshake for the SRP protocol
 * @param {string} identifier - Username/email address (this app uses emails)
 * @param {string} password - Self explanatory
 * @returns {string} the shared key generated by way of the SRP handshake
 * @throws {Error} on error from auth API or mismatch between client and server proofs
 * */
const loginUser = async function(identifier, password) {
  const a = await client.generateRandomSecret()
  const pub = await client.generatePublic(a)

  let A = pub.toString(16)
  if (A.length % 2 !== 0) {
    A = `0${A}`
  }

  const identityResp = await fetch("/api/auth/identify", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      "identifier": identifier,
      "A": A,
    }),
  })

  const idData = await identityResp.json()
  if (identityResp.status !== 200) {
    throw new Error(idData.error)
  }

  console.log(idData)

  const { salt, B } = idData
  const x = await client.generatePasswordHash(hexToBuf(salt), identifier, password)

  const { key, proof } = await client.step2(identifier, hexToBuf(salt), pub, a, hexToBigint(B), x)
  console.log(key, proof)

  const loginResp = await fetch("/api/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      "identifier": identifier,
      "proof": proof.toString(16),
    }),
  })

  const loginData = await loginResp.json()
  if (loginResp.status !== 200) {
    throw new Error(loginData.error)
  }

  console.log(loginData)
  const serverProof = loginData['proof']

  await client.step3(pub, key, proof, serverProof)

  return key.toString(16)
}

export { registerUser, loginUser }