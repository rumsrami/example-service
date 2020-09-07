/* tslint:disable */
// chat 0.0.1 a4fdc6d6993d7313905a53c2e256053b6a0b3ee2
// --
// This file has been generated by https://github.com/webrpc/webrpc using gen/typescript
// Do not edit by hand. Update your webrpc schema and re-generate.

// WebRPC description and code-gen version
export const WebRPCVersion = "v1"

// Schema version of your RIDL schema
export const WebRPCSchemaVersion = "0.0.1"

// Schema hash generated from your RIDL schema
export const WebRPCSchemaHash = "a4fdc6d6993d7313905a53c2e256053b6a0b3ee2"


//
// Types
//
export interface Version {
  webrpcVersion: string
  schemaVersion: string
  schemaHash: string
  appVersion: string
}

export interface ChatMessage {
  fromEmail: string
  toEmail: string
  messageUUID: string
  PK: string
  SK: string
  messageText: string
  seen: boolean
  delivered: boolean
  updatedAt?: string
  version: string
}

export interface Chat {
  ping(headers?: object): Promise<PingReturn>
  version(headers?: object): Promise<VersionReturn>
  createChatMessage(args: CreateChatMessageArgs, headers?: object): Promise<CreateChatMessageReturn>
}

export interface PingArgs {
}

export interface PingReturn {
  status: boolean  
}
export interface VersionArgs {
}

export interface VersionReturn {
  version: Version  
}
export interface CreateChatMessageArgs {
  req: ChatMessage
}

export interface CreateChatMessageReturn {
  res: boolean  
}


  
//
// Client
//
export class Chat implements Chat {
  private hostname: string
  private fetch: Fetch
  private path = '/rpc/Chat/'

  constructor(hostname: string, fetch: Fetch) {
    this.hostname = hostname
    this.fetch = fetch
  }

  private url(name: string): string {
    return this.hostname + this.path + name
  }
  
  ping = (headers?: object): Promise<PingReturn> => {
    return this.fetch(
      this.url('Ping'),
      createHTTPRequest({}, headers)
      ).then((res) => {
      return buildResponse(res).then(_data => {
        return {
          status: <boolean>(_data.status)
        }
      })
    })
  }
  
  version = (headers?: object): Promise<VersionReturn> => {
    return this.fetch(
      this.url('Version'),
      createHTTPRequest({}, headers)
      ).then((res) => {
      return buildResponse(res).then(_data => {
        return {
          version: <Version>(_data.version)
        }
      })
    })
  }
  
  createChatMessage = (args: CreateChatMessageArgs, headers?: object): Promise<CreateChatMessageReturn> => {
    return this.fetch(
      this.url('CreateChatMessage'),
      createHTTPRequest(args, headers)).then((res) => {
      return buildResponse(res).then(_data => {
        return {
          res: <boolean>(_data.res)
        }
      })
    })
  }
  
}

  
export interface WebRPCError extends Error {
  code: string
  msg: string
	status: number
}

const createHTTPRequest = (body: object = {}, headers: object = {}): object => {
  return {
    method: 'POST',
    headers: { ...headers, 'Content-Type': 'application/json' },
    body: JSON.stringify(body || {})
  }
}

const buildResponse = (res: Response): Promise<any> => {
  return res.text().then(text => {
    let data
    try {
      data = JSON.parse(text)
    } catch(err) {
      throw { code: 'unknown', msg: `expecting JSON, got: ${text}`, status: res.status } as WebRPCError
    }
    if (!res.ok) {
      throw data // webrpc error response
    }
    return data
  })
}

export type Fetch = (input: RequestInfo, init?: RequestInit) => Promise<Response>
