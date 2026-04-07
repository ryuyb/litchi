import { useForm } from "@tanstack/react-form";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Loader2, TerminalIcon } from "lucide-react";
import { toast } from "sonner";
import { z } from "zod";
import { usePostApiV1AuthLogin } from "#/api/auth/auth";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";

export const Route = createFileRoute("/login")({
	component: LoginPage,
});

const loginSchema = z.object({
	username: z.string().min(1, "Username is required"),
	password: z.string().min(1, "Password is required"),
});

function LoginPage() {
	const navigate = useNavigate();

	const loginMutation = usePostApiV1AuthLogin({
		mutation: {
			onSuccess: () => {
				toast.success("Login successful");
				navigate({ to: "/" });
			},
			onError: (_error) => {
				toast.error("Login failed. Please check your credentials.");
			},
		},
	});

	const form = useForm({
		defaultValues: {
			username: "",
			password: "",
		},
		validators: {
			onChange: loginSchema,
		},
		onSubmit: async ({ value }) => {
			loginMutation.mutate({
				data: { username: value.username, password: value.password },
			});
		},
	});

	return (
		<div className="flex min-h-screen w-full relative overflow-hidden items-center justify-center p-4">
			<div className="relative z-10 w-full max-w-md animate-slide-up-fade">
				<div className="glass-card rounded-3xl p-8 shadow-2xl backdrop-blur-xl">
					<div className="text-center mb-8 flex flex-col items-center">
						<div className="flex aspect-square size-16 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-primary/80 text-primary-foreground shadow-lg shadow-primary/20 mb-6 transition-transform hover:scale-105">
							<TerminalIcon className="size-8" />
						</div>
						<h1 className="text-3xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-foreground to-foreground/70">
							Welcome Back
						</h1>
						<p className="mt-2 text-muted-foreground">
							Sign in to your Litchi account
						</p>
					</div>

					<form
						onSubmit={(e) => {
							e.preventDefault();
							e.stopPropagation();
							form.handleSubmit();
						}}
						className="space-y-6"
					>
						<div className="space-y-4">
							<form.Field
								name="username"
								// biome-ignore lint/correctness/noChildrenProp: TanStack Form requires children as prop for render prop pattern
								children={(field) => (
									<div className="space-y-2 relative text-left">
										<Label
											htmlFor="username"
											className="text-foreground/80 font-medium"
										>
											Username
										</Label>
										<Input
											id="username"
											type="text"
											placeholder="Enter your username"
											value={field.state.value}
											onChange={(e) => field.handleChange(e.target.value)}
											onBlur={field.handleBlur}
											className={`bg-background/50 border-border/50 text-foreground placeholder:text-muted-foreground/70 rounded-xl h-12 px-4 focus-visible:ring-primary/50 focus-visible:border-primary ${field.state.meta.errors.length > 0 ? "border-destructive/50 focus-visible:ring-destructive/30" : ""}`}
											disabled={loginMutation.isPending}
										/>
										{field.state.meta.errors.length > 0 && (
											<p className="text-destructive text-sm font-medium flex items-center gap-1.5 animate-slide-up-fade">
												<span className="bg-destructive/20 text-destructive rounded-full w-4 h-4 flex items-center justify-center text-[10px] shrink-0">
													!
												</span>
												{field.state.meta.errors[0]?.message}
											</p>
										)}
									</div>
								)}
							/>

							<form.Field
								name="password"
								// biome-ignore lint/correctness/noChildrenProp: TanStack Form requires children as prop for render prop pattern
								children={(field) => (
									<div className="space-y-2 relative text-left">
										<div className="flex items-center justify-between">
											<Label
												htmlFor="password"
												className="text-foreground/80 font-medium"
											>
												Password
											</Label>
										</div>
										<Input
											id="password"
											type="password"
											placeholder="••••••••"
											value={field.state.value}
											onChange={(e) => field.handleChange(e.target.value)}
											onBlur={field.handleBlur}
											className={`bg-background/50 border-border/50 text-foreground placeholder:text-muted-foreground/70 rounded-xl h-12 px-4 focus-visible:ring-primary/50 focus-visible:border-primary ${field.state.meta.errors.length > 0 ? "border-destructive/50 focus-visible:ring-destructive/30" : ""}`}
											disabled={loginMutation.isPending}
										/>
										{field.state.meta.errors.length > 0 && (
											<p className="text-destructive text-sm font-medium flex items-center gap-1.5 animate-slide-up-fade">
												<span className="bg-destructive/20 text-destructive rounded-full w-4 h-4 flex items-center justify-center text-[10px] shrink-0">
													!
												</span>
												{field.state.meta.errors[0]?.message}
											</p>
										)}
									</div>
								)}
							/>
						</div>

						<form.Subscribe
							selector={(state) => [state.canSubmit, state.isSubmitting]}
							// biome-ignore lint/correctness/noChildrenProp: TanStack Form Subscribe requires children as prop for render prop pattern
							children={([canSubmit, isSubmitting]) => (
								<Button
									type="submit"
									className="w-full h-12 text-base font-bold rounded-xl shadow-lg shadow-primary/25 transition-all duration-300 transform active:scale-[0.98]"
									disabled={
										!canSubmit || isSubmitting || loginMutation.isPending
									}
								>
									{isSubmitting || loginMutation.isPending ? (
										<>
											<Loader2 className="mr-2 h-5 w-5 animate-spin" />
											Signing in...
										</>
									) : (
										"Sign In"
									)}
								</Button>
							)}
						/>
					</form>

					<div className="mt-8 text-center text-sm font-medium text-muted-foreground/70">
						Automated development agent system
					</div>
				</div>
			</div>
		</div>
	);
}
