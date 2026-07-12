import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { AuthButtonView } from "../AuthButton/AuthButton";
import Header from "./header";

const meta = {
  component: Header,
} satisfies Meta<typeof Header>;
export default meta;
type Story = StoryObj<typeof meta>;

export const LoggedOut: Story = {
  args: {
    authButton: (
      <AuthButtonView onLogin={() => undefined} onLogout={() => undefined} />
    ),
  },
};

export const LoggedIn: Story = {
  args: {
    authButton: (
      <AuthButtonView
        userName="Test User"
        userEmail="test@example.com"
        onLogin={() => undefined}
        onLogout={() => undefined}
      />
    ),
  },
};
