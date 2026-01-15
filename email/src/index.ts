import * as PostalMime from 'postal-mime';

export default {
	async email(message, env, ctx) {
		const parser = new PostalMime.default();
		const rawEmail = new Response(message.raw);
		const email = await parser.parse(await rawEmail.arrayBuffer());
		const form = new FormData();

		let username = email.from?.name || "Team Email";
		if (email.from?.address) username += ` <${email.from.address}>`;
		form.append("username", username);

		let msg = email.text || email.html || "(no content)";
		msg = msg.replaceAll(/\n--\s*\n\s*You received this message.*$/sg, "");
		msg = msg.trim();
		form.append("files[0]", new Blob([msg], { type: "text/plain" }), "message.txt");

		if (email.subject) {
			const content = `-# ${email.subject.replace("[🌊🎨🎡] ", "")}`;
			form.append("content", content);
		}

		const r = await fetch((env as any).DISCORD_URL, {
			method: "POST",
			body: form,
		});
		console.log(r.status, email.subject);
		if (r.status !== 200) {
			throw new Error(`failed to send webhook: ${await r.json()}`);
		}
	},
} satisfies ExportedHandler<Env>;
