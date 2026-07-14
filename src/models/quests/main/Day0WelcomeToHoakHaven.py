from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward


class Day0WelcomeToHoakHavenQuest(Quest):
    @property
    def id(self) -> str:
        return "day0_welcome"

    @property
    def title_key(self) -> str:
        return "quests.day0_welcome.title"

    @property
    def description_key(self) -> str:
        return "quests.day0_welcome.description"

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day0_welcome.step0_dialogue",
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.day0_welcome.step1_activity",
                rewards=QuestReward(xp=100),
                target_stat="daily_uses",
                target_count=1
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.day0_welcome.step2_dialogue",
                rewards=QuestReward(money=100)
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx != 1:
            return False

        step = self.steps[step_idx]
        if activity != step.extra.get('target_stat'):
            return False

        from src.database.quest import update_quest_progress
        new_val = current_progress.get('progress_value', 0) + amount
        update_quest_progress(user_id, self.id, progress_value=new_val)

        return new_val >= step.extra.get('target_count', 10)

    def get_unlocks(self) -> List[str]:
        return ["day1_strata"]
