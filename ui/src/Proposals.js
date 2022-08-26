
const HOST = '/api/v3/proposals'
export async function Proposals() {
    const resposne = await fetch(HOST)
    if (resposne.status !== 200) {
        console.error(resposne.status, resposne.body)
        return
    }
    return await resposne.json()
}