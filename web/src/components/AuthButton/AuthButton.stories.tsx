import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { AuthButtonView } from "./AuthButton";

const meta = {
  component: AuthButtonView,
  args: {
    onLogin: () => undefined,
    onLogout: () => undefined,
  },
} satisfies Meta<typeof AuthButtonView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const LoggedOut: Story = {};

export const LoggedIn: Story = {
  args: {
    userName: "Test User",
    userEmail: "test@example.com",
  },
};
