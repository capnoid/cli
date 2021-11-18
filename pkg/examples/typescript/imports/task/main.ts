// Linked to https://app.airplane.dev/t/typescript_imports [do not edit this line]

import airplane from 'airplane'
import { uppercase } from '../lib/text'

type Params = {
  id: string
}

export default async function(params: Params) {
  airplane.output(uppercase("your id:"))
  airplane.output(params.id)
}
